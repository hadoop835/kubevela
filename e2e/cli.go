package e2e

import (
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var rudrPath = GetCliBinary()

//GetCliBinary is to build kubevela binary.
func GetCliBinary() string {
	cwd, _ := os.Getwd()
	return path.Join(cwd, "../..", "./bin")
}

func Exec(cli string) (string, error) {
	var output []byte
	session, err := AsyncExec(cli)
	if err != nil {
		return string(output), err
	}
	s := session.Wait(30 * time.Second)
	return string(s.Out.Contents()) + string(s.Err.Contents()), nil
}
func ExecAndTerminate(cli string) (string, error) {
	var output []byte
	session, err := AsyncExec(cli)
	if err != nil {
		return string(output), err
	}
	time.Sleep(3 * time.Second)
	s := session.Terminate()
	return string(s.Out.Contents()) + string(s.Err.Contents()), nil
}

func LongTimeExec(cli string, timeout time.Duration) (string, error) {
	var output []byte
	session, err := AsyncExec(cli)
	if err != nil {
		return string(output), err
	}
	s := session.Wait(timeout)
	return string(s.Out.Contents()) + string(s.Err.Contents()), nil
}

func AsyncExec(cli string) (*gexec.Session, error) {
	c := strings.Fields(cli)
	commandName := path.Join(rudrPath, c[0])
	command := exec.Command(commandName, c[1:]...)
	session, err := gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	return session, err
}

func InteractiveExec(cli string, consoleFn func(*expect.Console)) (string, error) {
	var output []byte
	console, _, err := vt10x.NewVT10XConsole(expect.WithStdout(ginkgo.GinkgoWriter))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	defer console.Close()
	doneC := make(chan struct{})

	go func() {
		defer ginkgo.GinkgoRecover()
		defer close(doneC)
		consoleFn(console)
	}()

	c := strings.Fields(cli)
	commandName := path.Join(rudrPath, c[0])
	command := exec.Command(commandName, c[1:]...)
	command.Stdin = console.Tty()

	session, err := gexec.Start(command, console.Tty(), console.Tty())
	s := session.Wait(90 * time.Second)
	console.Tty().Close()
	<-doneC
	if err != nil {
		return string(output), err
	}
	return string(s.Out.Contents()) + string(s.Err.Contents()), nil
}

func BeforeSuit() {
	//Without this line, will hit issue like `<string>: Error: unknown command "scale" for "vela"`
	_, _ = Exec("vela system update")
}

func newK8sClient() (client.Client, error) {
	conf, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	scheme := k8sruntime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := oamcore.AddToScheme(scheme); err != nil {
		return nil, err
	}

	k8sclient, err := client.New(conf, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return k8sclient, nil
}
