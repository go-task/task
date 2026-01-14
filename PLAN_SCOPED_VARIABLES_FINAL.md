# Plan : Scoped Includes (Lazy DAG)

## Objectif

Scoper les variables des Taskfiles inclus via **lazy resolution** sur le DAG, avec **isolation stricte**.

## Experiment Flag

```bash
TASK_X_SCOPED_INCLUDES=1
```

---

## Modèle de scopes

### Priorité (croissante)

```
1. Environment      ← Shell + CLI (task FOO=bar)
2. Root vars        ← Taskfile racine
3. Include vars     ← Chaîne d'héritage (parent → enfant)
4. Task vars        ← Plus haut
```

### Visibilité (isolation stricte)

| Depuis | Voit | Ne voit PAS |
|--------|------|-------------|
| Root | Ses vars | Vars des includes |
| Include | Ses vars + héritage parent | Vars des siblings |
| Task | Toute la chaîne d'héritage | - |

### Partage de variables

Pour partager des vars entre plusieurs Taskfiles, utiliser `flatten: true` :

```yaml
includes:
  common:
    taskfile: ../common/Taskfile.yml
    flatten: true  # Merge global, pas de scoping
```

---

## Considérations spéciales

### Variables dynamiques (`sh:`)

```yaml
vars:
  VERSION:
    sh: git describe --tags
```

- Exécutées dans le `Dir` de leur Taskfile d'origine
- Le champ `Var.Dir` stocke le répertoire d'exécution
- Cache des résultats pour éviter les re-exécutions

### Defer

- Utilise le même cache de variables résolu
- Pas d'impact sur l'implémentation

### Ordre de résolution

```yaml
vars:
  A: "hello"
  B: "{{.A}} world"
  C:
    sh: echo "{{.B}}"
```

- Variables résolues dans l'ordre de déclaration
- Chaîne d'héritage résolue AVANT les vars locales

---

## Architecture

### Avant (merge global)

```
Reader.Read() → TaskfileGraph
    ↓
TaskfileGraph.Merge() → Taskfile unique
    - Vars mergées globalement (last-one-wins)
    ↓
Compiler.TaskfileVars = Toutes les vars mergées
```

### Après (lazy DAG)

```
Reader.Read() → TaskfileGraph
    ↓
TaskfileGraph.Merge() → Taskfile racine
    - Vars NON mergées (restent dans le DAG)
    - Tasks mergées (comme avant)
    ↓
Executor.Graph = DAG préservé
    ↓
Compiler.getVariablesLazy(task) → Traverse le DAG
```

---

## Implémentation

### Étape 1 : Experiment flag

**Fichier** : `experiments/experiments.go`

```go
var ScopedIncludes Experiment

func ParseWithConfig(dir string, config *ast.TaskRC) {
    // ...
    ScopedIncludes = New("SCOPED_INCLUDES", config, 1)
}
```

### Étape 2 : Stocker le DAG dans l'Executor

**Fichier** : `executor.go`

```go
type Executor struct {
    // ...
    Graph *ast.TaskfileGraph
}
```

**Fichier** : `setup.go`

```go
func (e *Executor) readTaskfile(node taskfile.Node) error {
    graph, err := reader.Read(ctx, node)
    e.Graph = graph
    e.Taskfile, err = graph.Merge()
    return nil
}
```

### Étape 3 : Ne pas merger les vars (si experiment ON)

**Fichier** : `taskfile/ast/taskfile.go`

```go
func (t1 *Taskfile) Merge(t2 *Taskfile, include *Include, experimentEnabled bool) error {
    if !experimentEnabled || include.Flatten {
        // Legacy ou flatten : merge global
        t1.Vars.Merge(t2.Vars, include)
        t1.Env.Merge(t2.Env, include)
    }
    // Sinon : vars restent dans le DAG

    return t1.Tasks.Merge(t2.Tasks, include, t1.Vars)
}
```

### Étape 4 : Helpers pour le DAG

**Fichier** : `taskfile/ast/graph.go`

```go
func (tfg *TaskfileGraph) Root() (*TaskfileVertex, error)

func (tfg *TaskfileGraph) GetVertexByNamespace(namespace string) (*TaskfileVertex, error)
```

### Étape 5 : Résolution lazy dans le Compiler

**Fichier** : `compiler.go`

```go
type Compiler struct {
    // ...
    Graph     *ast.TaskfileGraph
    varsCache map[string]*ast.Vars  // Cache par namespace
}

func (c *Compiler) getVariables(t *ast.Task, call *Call, eval bool) (*ast.Vars, error) {
    if experiments.ScopedIncludes.Enabled() {
        return c.getVariablesLazy(t, call, eval)
    }
    // Legacy...
}

func (c *Compiler) getVariablesLazy(t *ast.Task, call *Call, eval bool) (*ast.Vars, error) {
    result := env.GetEnviron()

    // 1. Special vars
    // 2. Root vars (depuis DAG.Root())
    // 3. Include chain vars (traverse DAG selon t.Namespace)
    // 4. CLI vars (call.Vars)
    // 5. Task vars (t.Vars)

    return result, nil
}
```

---

## Fichiers à modifier

| Fichier | Changement |
|---------|------------|
| `experiments/experiments.go` | Ajouter `ScopedIncludes` |
| `executor.go` | Ajouter `Graph` |
| `setup.go` | Stocker le DAG |
| `taskfile/ast/taskfile.go` | Ne pas merger vars si experiment ON |
| `taskfile/ast/graph.go` | Helpers `Root()`, `GetVertexByNamespace()` |
| `compiler.go` | `getVariablesLazy()` + cache |

---

## Tests

### Variables
1. Héritage : include voit vars du parent
2. Override : include peut override une var du parent
3. Isolation : parent ne voit PAS vars de l'include
4. Siblings : includes ne se voient pas entre eux
5. Chaîne : root → a → b fonctionne
6. Flatten : `flatten: true` = merge global
7. Legacy : flag OFF = comportement inchangé

### Variables dynamiques
8. `sh:` exécuté dans le Dir de l'include
9. Var dynamique référençant une var héritée
10. Cache des résultats

### Defer
11. Defer a accès aux vars scopées

---

## Ordre des commits

1. `feat(experiments): add SCOPED_INCLUDES experiment`
2. `feat(executor): store TaskfileGraph for lazy resolution`
3. `feat(graph): add Root() and GetVertexByNamespace() helpers`
4. `feat(taskfile): skip var merge when SCOPED_INCLUDES enabled`
5. `feat(compiler): implement lazy variable resolution`
6. `test: add scoped includes variable tests`
