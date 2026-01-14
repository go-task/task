# Plan : Scoped Variables pour les Includes de Taskfiles

## Objectif

Implémenter le scoping des variables pour les Taskfiles inclus. Les variables d'un include seront accessibles via `{{.namespace.VAR}}` au lieu d'être mergées globalement (problème actuel de conflits).

## Décisions prises

- **Accès** : Préfixe namespace (`{{.db.HOST}}`)
- **Conflits** : Préfixage obligatoire, pas de merge global
- **Requires/Preconditions** : Reporté à une phase ultérieure

---

## Visibilité des variables

| Relation | Accès | Mécanisme |
|----------|-------|-----------|
| Parent → Include | `{{.ns.VAR}}` | Le but principal |
| Include → Parent | Non (via `includes.vars:` si besoin) | Isolation pour réutilisabilité |
| Include → Sibling | Non (via parent comme passeur) | Évite les dépendances implicites |
| Include → Soi-même | `{{.VAR}}` | Naturel |

**Exemple de passage explicite du parent vers l'include :**
```yaml
includes:
  db:
    taskfile: ./db/Taskfile.yml
    vars:
      HOST: "{{.MY_HOST}}"  # Passage explicite des vars nécessaires
```

**Exemple de passage entre siblings via le parent :**
```yaml
includes:
  a: ./a/Taskfile.yml
  b:
    taskfile: ./b/Taskfile.yml
    vars:
      A_VAR: "{{.a.SOME_VAR}}"  # Parent passe les vars de a à b
```

---

## Étape 1 : Ajouter l'experiment flag

**Fichier** : `experiments/experiments.go`

```go
var (
    GentleForce       Experiment
    RemoteTaskfiles   Experiment
    EnvPrecedence     Experiment
    ScopedVariables   Experiment  // NEW
)

func ParseWithConfig(dir string, config *ast.TaskRC) {
    // ...
    ScopedVariables = New("SCOPED_VARIABLES", config, 1)
}
```

Activation : `TASK_X_SCOPED_VARIABLES=1`

---

## Étape 2 : Modifier la structure `Vars`

**Fichier** : `taskfile/ast/vars.go`

Ajouter un champ pour stocker les variables scopées par namespace :

```go
type Vars struct {
    om          *orderedmap.OrderedMap[string, Var]
    mutex       sync.RWMutex
    scoped      map[string]*Vars  // NEW: namespace -> vars
    scopedMutex sync.RWMutex
}
```

Nouvelles méthodes à ajouter :

```go
// SetScoped stocke les variables d'un namespace
func (vars *Vars) SetScoped(namespace string, scopedVars *Vars)

// GetScoped récupère les variables d'un namespace
func (vars *Vars) GetScoped(namespace string) (*Vars, bool)

// MergeScoped merge les variables dans un namespace au lieu de globalement
func (vars *Vars) MergeScoped(namespace string, other *Vars, include *Include)
```

Modifier `ToCacheMap()` pour inclure les namespaces comme maps imbriquées :

```go
func (vars *Vars) ToCacheMap() map[string]any {
    m := make(map[string]any, vars.Len())
    // Variables plates existantes
    for k, v := range vars.All() { ... }

    // NEW: Ajouter les namespaces comme maps imbriquées
    for namespace, scopedVars := range vars.scoped {
        m[namespace] = scopedVars.ToCacheMap()
    }
    return m
}
```

Cela permet `{{.db.HOST}}` naturellement via Go templates.

---

## Étape 3 : Modifier `Taskfile.Merge()`

**Fichier** : `taskfile/ast/taskfile.go`

```go
func (t1 *Taskfile) Merge(t2 *Taskfile, include *Include) error {
    // ... validations existantes ...

    if experiments.ScopedVariables.Enabled() && include != nil && !include.Flatten {
        // Scoped merge : variables dans le namespace
        t1.Vars.MergeScoped(include.Namespace, t2.Vars, include)
        t1.Env.MergeScoped(include.Namespace, t2.Env, include)
    } else {
        // Comportement legacy : merge plat
        t1.Vars.Merge(t2.Vars, include)
        t1.Env.Merge(t2.Env, include)
    }

    return t1.Tasks.Merge(t2.Tasks, include, t1.Vars)
}
```

---

## Étape 4 : Accès local sans préfixe pour les tasks de l'include

**Fichier** : `compiler.go`

Dans `getVariables()`, ajouter les variables du namespace courant pour qu'une task de l'include puisse y accéder sans préfixe :

```go
// Si la task a un namespace, ajouter ses variables localement
if experiments.ScopedVariables.Enabled() && t != nil && t.Namespace != "" {
    if scopedVars, ok := c.TaskfileVars.GetScoped(t.Namespace); ok {
        for k, v := range scopedVars.All() {
            result.Set(k, v)  // Accessible via {{.VAR}} dans le namespace
        }
    }
}
```

---

## Étape 5 : Gérer les includes imbriqués

**Fichier** : `taskfile/ast/graph.go`

Pour les includes imbriqués (a→b→c), propager le path complet du namespace :
- `{{.a.VAR}}` - variable de a
- `{{.a.b.VAR}}` - variable de b (inclus par a)

Option : ajouter `FullNamespacePath []string` dans `Include` ou calculer lors du merge.

---

## Étape 6 : Gérer le cas `flatten: true`

Déjà couvert par la condition `!include.Flatten` dans l'étape 3. Les includes aplatis conservent le comportement actuel (merge global).

---

## Étape 7 : Tests

**Nouveau fichier** : `testdata/includes/scoped_vars/`

Scénarios de test :
1. **Accès cross-namespace** : `{{.db.HOST}}` depuis le Taskfile parent
2. **Accès local** : `{{.HOST}}` depuis une task de l'include `db`
3. **Override via include.vars** : Les vars passées dans `includes:` continuent de fonctionner
4. **Pas de conflits** : Deux includes avec même nom de variable ne se marchent pas dessus
5. **Flatten** : `flatten: true` conserve le merge global
6. **Includes imbriqués** : `{{.a.b.VAR}}`

---

## Fichiers à modifier

| Fichier | Changement |
|---------|------------|
| `experiments/experiments.go` | Ajouter `ScopedVariables` experiment |
| `taskfile/ast/vars.go` | Ajouter `scoped` map, `MergeScoped()`, modifier `ToCacheMap()` |
| `taskfile/ast/taskfile.go` | Conditionner le merge scoped/plat |
| `compiler.go` | Injecter les vars du namespace dans le scope local de la task |
| `taskfile/ast/graph.go` | (optionnel) Propager le path namespace pour les includes imbriqués |

---

## Stratégie de migration

1. **v3.x** : Feature derrière `TASK_X_SCOPED_VARIABLES=1`
2. **v4.0** : Activer par défaut (breaking change documenté)
3. **Migration** : Mettre à jour les templates `{{.VAR}}` → `{{.namespace.VAR}}`

---

## Risques et considérations

- **Breaking change** : Les Taskfiles existants utilisant des variables d'includes sans préfixe casseront
- **Performance** : Négligeable (ajout d'une map)
- **Complexité** : Modérée, mais bien isolée dans quelques fichiers
