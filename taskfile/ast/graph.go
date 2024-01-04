package ast

import (
	"fmt"
	"os"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"golang.org/x/sync/errgroup"

	"github.com/go-task/task/v3/internal/filepathext"
)

type TaskfileGraph struct {
	graph.Graph[string, *TaskfileVertex]
}

// A TaskfileVertex is a vertex on the Taskfile DAG.
type TaskfileVertex struct {
	URI      string
	Taskfile *Taskfile
}

func taskfileHash(vertex *TaskfileVertex) string {
	return vertex.URI
}

func NewTaskfileGraph() *TaskfileGraph {
	return &TaskfileGraph{
		graph.New(taskfileHash,
			graph.Directed(),
			graph.PreventCycles(),
			graph.Rooted(),
		),
	}
}

func (tfg *TaskfileGraph) Visualize(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return draw.DOT(tfg.Graph, f)
}

func (tfg *TaskfileGraph) Merge() (*Taskfile, error) {
	hashes, err := graph.TopologicalSort(tfg.Graph)
	if err != nil {
		return nil, err
	}

	predecessorMap, err := tfg.PredecessorMap()
	if err != nil {
		return nil, err
	}

	for i := len(hashes) - 1; i >= 0; i-- {
		hash := hashes[i]

		// Get the current vertex
		vertex, err := tfg.Vertex(hash)
		if err != nil {
			return nil, err
		}

		// Create an error group to wait for all the included Taskfiles to be merged with all its parents
		var g errgroup.Group

		// Loop over each adjacent edge
		for _, edge := range predecessorMap[hash] {

			// TODO: Enable goroutines
			// Start a goroutine to process each included Taskfile
			// g.Go(
			err := func() error {
				// Get the child vertex
				predecessorVertex, err := tfg.Vertex(edge.Source)
				if err != nil {
					return err
				}

				// Get the merge options
				include, ok := edge.Properties.Data.(*Include)
				if !ok {
					return fmt.Errorf("task: Failed to get merge options")
				}

				// Handle advanced imports
				// i.e. where additional data is given when a Taskfile is included
				if include.AdvancedImport {
					predecessorVertex.Taskfile.Vars.Range(func(k string, v Var) error {
						o := v
						o.Dir = include.Dir
						predecessorVertex.Taskfile.Vars.Set(k, o)
						return nil
					})
					predecessorVertex.Taskfile.Env.Range(func(k string, v Var) error {
						o := v
						o.Dir = include.Dir
						predecessorVertex.Taskfile.Env.Set(k, o)
						return nil
					})
					for _, task := range vertex.Taskfile.Tasks.Values() {
						task.Dir = filepathext.SmartJoin(include.Dir, task.Dir)
						if task.IncludeVars == nil {
							task.IncludeVars = &Vars{}
						}
						task.IncludeVars.Merge(include.Vars)
						task.IncludedTaskfileVars = vertex.Taskfile.Vars
					}
				}

				// Merge the included Taskfile into the parent Taskfile
				if err := predecessorVertex.Taskfile.Merge(
					vertex.Taskfile,
					include,
				); err != nil {
					return err
				}

				return nil
			}()
			if err != nil {
				return nil, err
			}
			// )
		}

		// Wait for all the go routines to finish
		if err := g.Wait(); err != nil {
			return nil, err
		}
	}

	// Get the root vertex
	rootVertex, err := tfg.Vertex(hashes[0])
	if err != nil {
		return nil, err
	}

	rootVertex.Taskfile.Tasks.Range(func(name string, task *Task) error {
		if task == nil {
			task = &Task{}
			rootVertex.Taskfile.Tasks.Set(name, task)
		}
		task.Task = name
		return nil
	})

	return rootVertex.Taskfile, nil
}
