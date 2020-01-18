package deepq_test

import (
	"testing"

	. "github.com/pbarker/go-rl/pkg/v1/agent/deepq"
	envv1 "github.com/pbarker/go-rl/pkg/v1/env"
	sphere "github.com/pbarker/go-rl/pkg/v1/env"
	"github.com/pbarker/logger"
	"github.com/stretchr/testify/require"
	"gorgonia.org/tensor"
)

func TestPolicy(t *testing.T) {

	// test that network converges to static values.
	s, err := sphere.NewLocalServer(sphere.GymServerConfig)
	require.NoError(t, err)
	defer s.Resource.Close()

	env, err := s.Make("CartPole-v0")
	require.NoError(t, err)

	p, err := NewPolicy(DefaultPolicyConfig, env)
	require.NoError(t, err)

	xShape := env.ObservationSpaceShape()[0]
	x := tensor.New(tensor.WithShape(xShape), tensor.WithBacking([]float32{0.051960364, 0.14512223, 0.12799974, 0.63951147}))

	yShape := envv1.PotentialsShape(env.ActionSpace)[0]
	y := tensor.New(tensor.WithShape(yShape), tensor.WithBacking([]float32{0.4484117, -0.09160687}))

	qv1, err := p.Predict(x)
	require.NoError(t, err)

	cost1 := p.CostNode.Value()

	err = p.Fit(x, y)
	require.NoError(t, err)

	for i := 0; i < 10000; i++ {
		// qv, err := p.Predict(x)
		// require.NoError(t, err)

		// logger.Info("yhat: ", qv)
		logger.Info("y: ", y)
		err = p.Fit(x, y)
		require.NoError(t, err)
	}

	logger.Info("qv1: ", qv1)
	logger.Info("cost1: ", cost1)
}