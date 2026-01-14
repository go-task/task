# Plan V3 : Lazy DAG avec Isolation Stricte

## Objectif

Implémenter le scoping des variables via **lazy resolution** et **isolation stricte** (pas de `{{.namespace.VAR}}`).

## Décisions

- **Architecture** : Lazy resolution via DAG
- **Isolation stricte** : Pas d'accès cross-namespace
- **Héritage** : Include hérite du parent, pas l'inverse
- **Experiment** : `TASK_X_SCOPED_VARIABLES=1`

---

## Modèle de scopes

```
Environment     →  Shell + CLI vars
    ↓ hérite
Entrypoint      →  Root Taskfile vars
    ↓ hérite
Include(s)      →  Vars de chaque include (dans l'ordre)
    ↓ hérite
Task            →  Vars de la task
```

Chaque scope hérite du précédent et peut override.

---

## Visibilité

| Depuis | Voit | Ne voit PAS |
|--------|------|-------------|
| Root | Ses vars | Vars des includes |
| Include | Ses vars + parent | Vars des siblings |
| Task | Toute la chaîne | - |

**Communication unidirectionnelle** : parent → enfant via `includes.vars:`

---

## Changements requis

### 1. Experiment flag

**Fichier** : `experiments/experiments.go`

```go
var ScopedVariables Experiment

func ParseWithConfig(dir string, config *ast.TaskRC) {
    ScopedVariables = New("SCOPED_VARIABLES", config, 1)
}
```

---

### 2. Préserver le DAG

**Fichier** : `executor.go`

```go
type Executor struct {
    Taskfile *ast.Taskfile
    Graph    *ast.TaskfileGraph  // NOUVEAU
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

---

### 3. Ne pas merger les vars

**Fichier** : `taskfile/ast/taskfile.go`

```go
func (t1 *Taskfile) Merge(t2 *Taskfile, include *Include) error {
    if !experiments.ScopedVariables.Enabled() {
        // Legacy : merge global
        t1.Vars.Merge(t2.Vars, include)
        t1.Env.Merge(t2.Env, include)
    }
    // Les vars restent dans le DAG, pas de merge

    return t1.Tasks.Merge(t2.Tasks, include, t1.Vars)
}
```

---

### 4. Résolution lazy dans le Compiler

**Fichier** : `compiler.go`

```go
type Compiler struct {
    Graph *ast.TaskfileGraph  // NOUVEAU
}

func (c *Compiler) getVariables(t *ast.Task, call *Call, evaluateShVars bool) (*ast.Vars, error) {
    if experiments.ScopedVariables.Enabled() {
        return c.getVariablesLazy(t, call, evaluateShVars)
    }
    // Legacy behavior...
}

func (c *Compiler) getVariablesLazy(t *ast.Task, call *Call, evaluateShVars bool) (*ast.Vars, error) {
    result := env.GetEnviron()

    // 1. Special vars
    specialVars, _ := c.getSpecialVars(t, call)

    // 2. Root taskfile vars (depuis le DAG)
    rootVertex := c.Graph.Root()
    for k, v := range rootVertex.Taskfile.Vars.All() {
        result.Set(k, v)
    }

    // 3. Include chain vars (traverser le DAG jusqu'à cette task)
    if t != nil && t.Namespace != "" {
        includeVars := c.resolveIncludeChain(t.Namespace)
        for k, v := range includeVars.All() {
            result.Set(k, v)
        }
    }

    // 4. Task vars
    if t != nil {
        for k, v := range t.Vars.All() {
            result.Set(k, v)
        }
    }

    return result, nil
}

func (c *Compiler) resolveIncludeChain(namespace string) *ast.Vars {
    // namespace = "a:b:taskname" → trouver les vars de a, puis b
    // Traverser le DAG en suivant les edges
}
```

---

### 5. Helper pour trouver un vertex par namespace

**Fichier** : `taskfile/ast/graph.go`

```go
func (tfg *TaskfileGraph) GetVertexByNamespace(namespace string) (*TaskfileVertex, error) {
    // Trouver le vertex correspondant au namespace
}

func (tfg *TaskfileGraph) Root() *TaskfileVertex {
    // Retourner le vertex racine
}
```

---

## Fichiers à modifier

| Fichier | Changement |
|---------|------------|
| `experiments/experiments.go` | Ajouter `ScopedVariables` |
| `executor.go` | Ajouter `Graph` |
| `setup.go` | Stocker le DAG |
| `taskfile/ast/taskfile.go` | Conditionner le merge |
| `taskfile/ast/graph.go` | Helpers pour accès au DAG |
| `compiler.go` | `getVariablesLazy()` |

---

## Tests

1. **Héritage** : Include voit les vars du parent
2. **Override** : Include peut override une var du parent
3. **Isolation** : Parent ne voit pas les vars de l'include
4. **Siblings** : Un include ne voit pas les vars d'un autre
5. **Chaîne** : `root → a → b` - b voit les vars de a et root
6. **Legacy** : Sans le flag, comportement inchangé

---

## Avantages

1. **Simple** : Pas de mapping cross-namespace
2. **Performance** : Lazy, résout que le nécessaire
3. **Clair** : Héritage linéaire, facile à comprendre
4. **Safe** : Experiment flag, rollback facile

---

## Inconvénient

Le parent ne peut pas "lire" les vars d'un include. Si nécessaire, il faudrait ajouter `{{.namespace.VAR}}` plus tard (V2).
