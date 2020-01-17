package model

import (
	"github.com/pbarker/go-rl/pkg/v1/common"
	"github.com/pbarker/logger"
	g "gorgonia.org/gorgonia"
)

// Model is a prediction model.
type Model interface {
	// Compile the model.
	Compile(...Opt) error

	// Predict x.
	Predict(x g.Value) (prediction g.Value, err error)

	// Fit x to y.
	Fit(x, y g.Value) error

	// Visualize the model.
	Visualize()

	// Graph returns the expression graph for the model.
	Graph() *g.ExprGraph
}

// Sequential model.
type Sequential struct {
	// Layers in the model.
	Layers *Chain

	graph   *g.ExprGraph
	x       *g.Node
	y       *g.Node
	costVal g.Value
	predVal g.Value

	costFn     CostFn
	optimizer  g.Solver
	prediction *g.Node
	vm         g.VM
}

// NewSequential returns a new sequential model.
func NewSequential(x, y g.Value) *Sequential {
	graph := g.NewGraph()

	xn := g.NewTensor(graph, x.Dtype(), len(x.Shape()), g.WithValue(x), g.WithName("x"))
	yn := g.NewTensor(graph, y.Dtype(), len(y.Shape()), g.WithValue(y), g.WithName("y"))
	return &Sequential{
		Layers: NewChain(),
		graph:  graph,
		x:      xn,
		y:      yn,
	}
}

// Opt is a model option.
type Opt func(Model)

// WithLoss uses a specific cost function with the model.
// Defaults to MSE.
func WithLoss(costFn CostFn) func(Model) {
	return func(m Model) {
		switch t := m.(type) {
		case *Sequential:
			t.costFn = costFn
		default:
			logger.Fatal("unknown model type")
		}
	}
}

// WithOptimizer uses a specific optimizer function.
// Defaults to Adam.
func WithOptimizer(optimizer g.Solver) func(Model) {
	return func(m Model) {
		switch t := m.(type) {
		case *Sequential:
			t.optimizer = optimizer
		default:
			logger.Fatal("unknown model type")
		}
	}
}

// AddLayer adds a layer.
func (s *Sequential) AddLayer(layer Layer) {
	s.Layers.Add(layer)
}

// AddLayers adds a number of layers.
func (s *Sequential) AddLayers(layers ...Layer) {
	for _, layer := range layers {
		s.Layers.Add(layer)
	}
}

// Compile the model.
func (s *Sequential) Compile(opts ...Opt) error {
	for _, opt := range opts {
		opt(s)
	}
	if s.costFn == nil {
		s.costFn = MeanSquaredError
	}
	if s.optimizer == nil {
		s.optimizer = g.NewAdamSolver()
	}
	s.Layers.Compile(s)

	prediction, err := s.Layers.Fwd(s.x)
	if err != nil {
		return err
	}
	s.prediction = prediction
	g.Read(prediction, &s.predVal)

	cost, err := s.costFn(prediction, s.y)
	if err != nil {
		return err
	}
	g.Read(cost, &s.costVal)

	_, err = g.Grad(cost, s.Layers.Learnables()...)
	if err != nil {
		return err
	}

	s.vm = g.NewTapeMachine(s.Graph(), g.BindDualValues(s.Layers.Learnables()...))
	return nil
}

// Predict x.
func (s *Sequential) Predict(x g.Value) (prediction g.Value, err error) {
	s.vm.Reset()
	err = g.Let(s.x, x)
	if err != nil {
		return prediction, err
	}
	err = s.vm.RunAll()
	if err != nil {
		return prediction, err
	}
	prediction = s.predVal
	s.vm.Reset()
	return
}

// Fit x to y.
func (s *Sequential) Fit(x, y g.Value) error {
	// TODO: need to create a separate training graph
	s.vm.Reset()
	g.Let(s.y, y)
	g.Let(s.x, x)
	err := s.vm.RunAll()
	if err != nil {
		return err
	}
	logger.Info("cost: ", s.costVal)
	grads := g.NodesToValueGrads(s.Layers.Learnables())
	s.optimizer.Step(grads)
	s.vm.Reset()
	return nil
}

// Visualize the model.
func (s *Sequential) Visualize() {
	common.Visualize(s.Graph())
}

// Graph returns the expression graph for the model.
func (s *Sequential) Graph() *g.ExprGraph {
	return s.graph
}
