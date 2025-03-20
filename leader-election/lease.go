/*
 * Copyright (c) 2024.  Mike Hudgins <mchudgins@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package leader_election

import (
	"context"
	"github.com/mchudgins/go/helper"
	"os"
	"sync"
	"time"

	"github.com/mchudgins/go/log"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)
import "k8s.io/client-go/tools/leaderelection"

func MonitorLease(logger *zap.Logger, wg *sync.WaitGroup,
	clientset *kubernetes.Clientset, namespace, leaseName, hostname string) (context.CancelFunc, error) {
	leaderElectionConfig := leaderelection.LeaderElectionConfig{
		Lock: &resourcelock.LeaseLock{
			LeaseMeta: metav1.ObjectMeta{
				Name:      leaseName,
				Namespace: namespace,
			},
			Client: clientset.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: hostname,
			},
		},
		LeaseDuration: 30 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: onStartedLeading,
			OnStoppedLeading: func() {
				logger.Info("no longer the leader")
			},
			OnNewLeader: func(identity string) {
				logger.Info("a new leader has been assigned",
					zap.String("leaderName", identity))
			},
		},
		ReleaseOnCancel: true,
	}

	_, err := leaderelection.NewLeaderElector(leaderElectionConfig)
	if err != nil {
		logger.Fatal("invalid leaderElectionConfig",
			zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	//	defer cancel()

	rlogger := logger.With(zap.String("goRoutine", "MonitorLease"))
	ctx = log.NewContext(ctx, rlogger)

	helper.LaunchGoRoutine(rlogger, wg, func() {
		leaderelection.RunOrDie(ctx, leaderElectionConfig)
	})

	return cancel, nil
}

func onStartedLeading(ctx context.Context) {
	logger := log.FromContext(ctx)

	hostname := os.Getenv("POD_NAME")

	logger.Info("leading",
		zap.String("podName", hostname))

	// do initial stuff here to assume leadership....

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("stopped leader loop",
					zap.String("podName", hostname))
				return

			case <-time.After(5 * time.Second):
				logger.Info("still the leader",
					zap.String("podName", hostname))
			}
		}
	}()

}

func getKubeClient() (*kubernetes.Clientset, error) {
	// Create a Kubernetes client using the current context
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
