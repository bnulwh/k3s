package agent

import (
	"context"
	"math/rand"
	"time"

	"github.com/rancher/rio/pkg/daemons/config"

	"github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/util/logs"
	app2 "k8s.io/kubernetes/cmd/kube-proxy/app"
	"k8s.io/kubernetes/cmd/kubelet/app"
	_ "k8s.io/kubernetes/pkg/client/metrics/prometheus" // for client metric registration
	_ "k8s.io/kubernetes/pkg/version/prometheus"        // for version metric registration
)

func Agent(config *config.Agent) error {
	rand.Seed(time.Now().UTC().UnixNano())

	prepare(config)

	kubelet(config)
	kubeProxy(config)

	return nil
}

func prepare(config *config.Agent) {
	if config.CNIBinDir == "" {
		config.CNIBinDir = "/opt/cni/bin"
	}
	if config.CNIConfDir == "" {
		config.CNIConfDir = "/etc/cni/net.d"
	}
}

func kubeProxy(config *config.Agent) {
	args := []string{
		"--proxy-mode", "iptables",
		"--healthz-bind-address", "127.0.0.1",
		"--kubeconfig", config.KubeConfig,
		"--cluster-cidr", config.ClusterCIDR.String(),
	}
	args = append(args, config.ExtraKubeletArgs...)

	command := app2.NewProxyCommand()
	command.SetArgs(args)
	go func() {
		err := command.Execute()
		logrus.Fatalf("kube-proxy exited: %v", err)
	}()
}

func kubelet(config *config.Agent) {
	command := app.NewKubeletCommand(context.Background().Done())
	logs.InitLogs()
	defer logs.FlushLogs()

	args := []string{
		"--healthz-bind-address", "127.0.0.1",
		"--read-only-port", "0",
		"--allow-privileged=true",
		"--cluster-domain", "cluster.local",
		"--kubeconfig", config.KubeConfig,
		"--eviction-hard", "imagefs.available<5%,nodefs.available<5%",
		"--eviction-minimum-reclaim", "imagefs.available=10%,nodefs.available=10%",
		"--feature-gates=MountPropagation=true",
		"--node-ip", config.NodeIP,
		"--fail-swap-on=false",
		"--cgroup-root", "/k3s",
		"--cgroup-driver", "cgroupfs",
		"--cni-conf-dir", config.CNIConfDir,
		"--cni-bin-dir", config.CNIBinDir,
	}
	if len(config.ClusterDNS) > 0 {
		args = append(args, "--cluster-dns", config.ClusterDNS.String())
	}
	if config.RuntimeSocket != "" {
		args = append(args, "--container-runtime-endpoint", config.RuntimeSocket)
	}
	if config.ListenAddress != "" {
		args = append(args, "--address", config.ListenAddress)
	}
	if config.CACertPath != "" {
		args = append(args, "--anonymous-auth=false", "--client-ca-file", config.CACertPath)
	}
	if config.NodeName != "" {
		args = append(args, "--hostname-override", config.NodeName)
	}
	args = append(args, config.ExtraKubeletArgs...)

	command.SetArgs(args)

	go func() {
		logrus.Fatalf("kubelet exited: %v", command.Execute())
	}()
}
