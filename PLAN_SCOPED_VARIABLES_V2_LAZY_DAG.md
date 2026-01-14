# Plan V2 : Scoped Variables avec Lazy DAG

## Objectif

Implémenter le scoping des variables pour les Taskfiles inclus avec **lazy resolution via DAG** et **accès cross-namespace**.

## Décisions prises

- **Architecture** : Lazy resolution via DAG (pas de merge global des variables)
- **Accès cross-namespace** : `{{.namespace.VAR}}` pour accéder aux variables d'un include
- **Isolation** : Un include ne voit pas les vars du parent (sauf passage explicite via `includes.vars:`)
- **Requires/Preconditions** : Reporté à une phase ultérieure

---

## Comparaison des approches

| Aspect | Plan V1 (scoped merge) | Plan V2 (lazy DAG) |
|--------|------------------------|---------------------|
| Merge des vars | Oui, dans map scopée | Non, préservées dans le DAG |
| Résolution | Build time | Runtime (lazy) |
| Performance | Résout tout | Résout seulement le nécessaire |
| Complexité | Modérée | Plus élevée |
| Accès cross-namespace | `{{.ns.VAR}}` via map | `{{.ns.VAR}}` via traversée DAG |

---

## Modèle de scopes (priorité croissante)

```
1. Environment Scope (plus bas)
   - Variables d'environnement shell
   - Variables CLI (task VAR=value)

2. Entrypoint Scope
   - Dotenv globaux du Taskfile racine
   - Vars globales du Taskfile racine

3. Include Scope(s) (dans l'ordre d'inclusion)
   - Variables passées via `includes.vars:`
   - Dotenv de l'include (futur)
   - Vars globales de l'include

4. Task Scope (plus haut)
   - Variables passées via une autre task
   - Dotenv au niveau task
   - Vars au niveau task
```

**Règle** : Chaque scope hérite du précédent et peut override.

---

## Visibilité des variables

| Depuis... | Accès à... | Mécanisme |
|-----------|------------|-----------|
| Parent | Ses propres vars | `{{.VAR}}` |
| Parent | Vars d'un include | `{{.namespace.VAR}}` (traversée DAG) |
| Include | Ses propres vars | `{{.VAR}}` |
| Include | Vars du parent | Non (via `includes.vars:` si besoin) |
| Include | Vars d'un sibling | Non (via parent comme passeur) |

---

## Architecture technique

### Avant (merge global)

```
Reader.Read() → TaskfileGraph
    ↓
TaskfileGraph.Merge() → Taskfile unique
    - Vars mergées globalement (last-one-wins)
    - Tasks mergées avec namespace
    ↓
Executor.Taskfile = Taskfile unique
    ↓
Compiler.TaskfileVars = Taskfile.Vars (toutes mergées)
```

### Après (lazy DAG)

```
Reader.Read() → TaskfileGraph
    ↓
TaskfileGraph.Merge() → Taskfile racine
    - Vars NON mergées (restent dans chaque vertex du DAG)
    - Tasks mergées avec namespace (comme avant)
    ↓
Executor.Taskfile = Taskfile racine
Executor.Graph = TaskfileGraph (NOUVEAU)
    ↓
Compiler.Graph = TaskfileGraph
Compiler.resolveVariables(task) → traverse le DAG lazy
```

---

## Étapes d'implémentation

### Étape 1 : Ajouter l'experiment flag

**Fichier** : `experiments/experiments.go`

```go
var ScopedVariables Experiment

func ParseWithConfig(dir string, config *ast.TaskRC) {
    ScopedVariables = New("SCOPED_VARIABLES", config, 1)
}
```

---

### Étape 2 : Préserver le DAG dans l'Executor

**Fichier** : `executor.go` + `setup.go`

```go
type Executor struct {
    // ... existant ...
    Taskfile *ast.Taskfile
    Graph    *ast.TaskfileGraph  // NOUVEAU
}

func (e *Executor) readTaskfile(node taskfile.Node) error {
    graph, err := reader.Read(ctx, node)
    e.Graph = graph  // Stocker le DAG
    e.Taskfile, err = graph.Merge()  // Merge pour les tasks
    return nil
}
```

---

### Étape 3 : Modifier le merge pour ne pas merger les variables

**Fichier** : `taskfile/ast/taskfile.go`

```go
func (t1 *Taskfile) Merge(t2 *Taskfile, include *Include) error {
    // ... validations existantes ...

    if experiments.ScopedVariables.Enabled() {
        // NE PAS merger les variables - elles restent dans le DAG
        // Optionnel : merger seulement les vars passées via include.Vars
    } else {
        // Comportement legacy
        t1.Vars.Merge(t2.Vars, include)
        t1.Env.Merge(t2.Env, include)
    }

    return t1.Tasks.Merge(t2.Tasks, include, t1.Vars)
}
```

---

### Étape 4 : Passer le DAG au Compiler

**Fichier** : `compiler.go`

```go
type Compiler struct {
    // ... existant ...
    Graph *ast.TaskfileGraph  // NOUVEAU
}
```

---

### Étape 5 : Résolution lazy des variables

**Fichier** : `compiler.go`

Nouvelle méthode pour résoudre les variables en traversant le DAG :

```go
func (c *Compiler) resolveVariablesFromGraph(t *ast.Task) (*ast.Vars, error) {
    result := env.GetEnviron()

    // 1. Variables spéciales
    specialVars, _ := c.getSpecialVars(t, call)

    // 2. Variables du Taskfile racine (entrypoint scope)
    rootVars := c.getRootTaskfileVars()

    // 3. Variables de la chaîne d'includes jusqu'à cette task (include scope)
    //    Traverser le DAG en suivant le namespace de la task
    includeChainVars := c.getIncludeChainVars(t.Namespace)

    // 4. Variables de la task elle-même (task scope)
    taskVars := t.Vars

    // Merger dans l'ordre de priorité
    // ...

    return result, nil
}

func (c *Compiler) getIncludeChainVars(namespace string) *ast.Vars {
    // Trouver le vertex de l'include dans le DAG
    // Récupérer ses variables
    // Récursivement remonter si includes imbriqués
}
```

---

### Étape 6 : Supporter `{{.namespace.VAR}}` dans le templater

**Fichier** : `internal/templater/templater.go`

Deux approches possibles :

**A) Pré-populer une map avec les namespaces accessibles :**
```go
func (c *Compiler) buildTemplateVars(t *ast.Task) map[string]any {
    m := make(map[string]any)

    // Variables locales
    for k, v := range localVars {
        m[k] = v
    }

    // Namespaces accessibles (depuis la perspective de cette task)
    for _, include := range c.getAccessibleIncludes(t) {
        m[include.Namespace] = c.getIncludeVarsAsMap(include)
    }

    return m
}
```

**B) Custom template function :**
```go
// {{ns "db" "HOST"}} au lieu de {{.db.HOST}}
funcMap := template.FuncMap{
    "ns": func(namespace, varName string) any {
        return c.resolveNamespacedVar(namespace, varName)
    },
}
```

**Recommandation** : Approche A pour garder la syntaxe `{{.db.HOST}}`

---

### Étape 7 : Gérer les includes imbriqués

Pour `root → a → b`, les variables doivent être accessibles comme :
- `{{.a.VAR}}` depuis root
- `{{.a.b.VAR}}` depuis root (via a)
- `{{.b.VAR}}` depuis a
- `{{.VAR}}` depuis b (ses propres vars)

Le namespace complet est déjà tracké dans `Task.Namespace` (e.g., `"a:b:taskname"`).

---

### Étape 8 : Tests

**Nouveau répertoire** : `testdata/includes/scoped_vars/`

Scénarios :
1. Accès cross-namespace : `{{.db.HOST}}` depuis root
2. Accès local : `{{.HOST}}` depuis l'include
3. Isolation : include ne voit pas les vars du parent
4. Override via `includes.vars:`
5. Pas de conflits entre includes
6. `flatten: true` conserve le comportement legacy
7. Includes imbriqués : `{{.a.b.VAR}}`
8. Variables dynamiques (`sh:`) dans un include

---

## Fichiers à modifier

| Fichier | Changement |
|---------|------------|
| `experiments/experiments.go` | Ajouter `ScopedVariables` |
| `executor.go` | Ajouter champ `Graph` |
| `setup.go` | Stocker le DAG dans l'Executor |
| `taskfile/ast/taskfile.go` | Conditionner le merge des vars |
| `compiler.go` | Traversée lazy du DAG, accès au Graph |
| `internal/templater/templater.go` | (optionnel) Support `{{.ns.VAR}}` |

---

## Stratégie de migration

1. **v3.x** : Feature derrière `TASK_X_SCOPED_VARIABLES=1`
2. **v4.0** : Activer par défaut (breaking change documenté)
3. **Migration** :
   - Les vars d'includes ne sont plus accessibles directement → utiliser `{{.namespace.VAR}}`
   - Les vars du parent ne sont plus visibles dans l'include → passer via `includes.vars:`

---

## Avantages de cette approche

1. **Performance** : Résolution lazy, on ne calcule que ce qui est nécessaire
2. **Clarté** : Chaque scope est explicite
3. **Pas de conflits** : Les variables ne se marchent plus dessus
4. **Réutilisabilité** : Un include peut être utilisé dans différents contextes
5. **Utilise l'existant** : Le DAG existe déjà, on l'exploite mieux

---

## Risques et considérations

- **Breaking change** : Important - comportement fondamentalement différent
- **Complexité** : La traversée du DAG est plus complexe que le merge
- **Debug** : Plus difficile de comprendre d'où vient une variable (ajouter du logging)
