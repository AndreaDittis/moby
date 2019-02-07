package daemon // import "github.com/docker/docker/daemon"

import (
	// "encoding/json"
	// "fmt"
	// "io/ioutil"
	// "os/exec"
	// "path/filepath"

	"github.com/docker/docker/container"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// func (daemon *Daemon) addCustomInit(c *container.Container, s *specs.Spec) error {
// 	// only add the custom init if it is specified and the container is running in its
// 	// own private pid namespace.  It does not make sense to add if it is running in the
// 	// host namespace or another container's pid namespace where we already have an init
// 	if c.HostConfig.PidMode.IsPrivate() {
// 		if (c.HostConfig.Init != nil && *c.HostConfig.Init) ||
// 			(c.HostConfig.Init == nil && daemon.configStore.Init) {
// 			s.Process.Args = append([]string{inContainerInitPath, "--", c.Path}, c.Args...)
// 			path := daemon.configStore.InitPath
// 			if path == "" {
// 				path, err = exec.LookPath(daemonconfig.DefaultInitBinary)
// 				if err != nil {
// 					return err
// 				}
// 			}
// 			s.Mounts = append(s.Mounts, specs.Mount{
// 				Destination: inContainerInitPath,
// 				Type:        "bind",
// 				Source:      path,
// 				Options:     []string{"bind", "ro"},
// 			})
// 		}
// 	}
// }

func setLinuxDomainname(c *container.Container, s *specs.Spec) {
	// There isn't a field in the OCI for the NIS domainname, but luckily there
	// is a sysctl which has an identical effect to setdomainname(2) so there's
	// no explicit need for runtime support.
	s.Linux.Sysctl = make(map[string]string)
	if c.Config.Domainname != "" {
		s.Linux.Sysctl["kernel.domainname"] = c.Config.Domainname
	}
}
