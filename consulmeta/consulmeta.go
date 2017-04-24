package consulmeta

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"path"

	"github.com/gliderlabs/registrator/bridge"
	consulapi "github.com/hashicorp/consul/api"
)

const DefaultInterval = "10s"

func init() {
	bridge.Register(new(Factory), "consulmeta")
}

func (r *ConsulMetaAdapter) interpolateService(script string, service *bridge.Service) string {
	withIp := strings.Replace(script, "$SERVICE_IP", service.Origin.HostIP, -1)
	withPort := strings.Replace(withIp, "$SERVICE_PORT", service.Origin.HostPort, -1)
	return withPort
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) bridge.RegistryAdapter {
	config := consulapi.DefaultConfig()
	if uri.Host != "" {
		config.Address = uri.Host
	}
	client, err := consulapi.NewClient(config)
	if err != nil {
		log.Fatal("consulmeta: ", uri.Scheme)
	}
	return &ConsulMetaAdapter{client: client, path: uri.Path}
}

type ConsulMetaAdapter struct {
	client *consulapi.Client
	path   string
}

// Ping will try to connect to consul by attempting to retrieve the current leader.
func (r *ConsulMetaAdapter) Ping() error {
	status := r.client.Status()
	leader, err := status.Leader()
	if err != nil {
		return err
	}
	log.Println("consulmeta: current leader ", leader)

	return nil
}

func (r *ConsulMetaAdapter) Register(service *bridge.Service) error {
	registration := new(consulapi.AgentServiceRegistration)
	registration.ID = service.ID
	registration.Name = service.Name
	registration.Port = service.Port
	registration.Tags = service.Tags
	registration.Address = service.IP
	registration.Check = r.buildCheck(service)

	var err error

	base_path := path.Join(r.path, service.Name)
	// Strip off leading forward slash
	// because that's not allowed by Consul's KV store
	if strings.HasPrefix(base_path, "/") {
		base_path = base_path[1:]
	}
	for k, v := range service.Attrs {
		path := path.Join(base_path, k)
		_, err = r.client.KV().Put(&consulapi.KVPair{Key: path, Value: []byte(v)}, nil)
	}
	if err != nil {
		log.Println("consulmeta: failed to register metadata for service:", err)
		return err
	}

	err = r.client.Agent().ServiceRegister(registration)
	if err != nil {
		log.Println("consulmeta: failed to register service:", err)
		return err
	}
	return err
}

func (r *ConsulMetaAdapter) buildCheck(service *bridge.Service) *consulapi.AgentServiceCheck {
	check := new(consulapi.AgentServiceCheck)
	if path := service.Attrs["check_http"]; path != "" {
		check.HTTP = fmt.Sprintf("http://%s:%d%s", service.IP, service.Port, path)
		if timeout := service.Attrs["check_timeout"]; timeout != "" {
			check.Timeout = timeout
		}
	} else if cmd := service.Attrs["check_cmd"]; cmd != "" {
		check.Script = fmt.Sprintf("check-cmd %s %s %s", service.Origin.ContainerID[:12], service.Origin.ExposedPort, cmd)
	} else if script := service.Attrs["check_script"]; script != "" {
		check.Script = r.interpolateService(script, service)
	} else if ttl := service.Attrs["check_ttl"]; ttl != "" {
		check.TTL = ttl
	} else {
		return nil
	}
	if check.Script != "" || check.HTTP != "" {
		if interval := service.Attrs["check_interval"]; interval != "" {
			check.Interval = interval
		} else {
			check.Interval = DefaultInterval
		}
	}
	return check
}

func (r *ConsulMetaAdapter) Deregister(service *bridge.Service) error {
	var err error

	err = r.client.Agent().ServiceDeregister(service.ID)
	if err != nil {
		log.Println("consulmeta: failed to deregister service:", err)
		return err
	}

	// Only remove KV data when there are no more services with the same name
	var services, _, _ = r.client.Catalog().Service(service.Name, "", nil)
	if len(services) == 0 {
		path := r.path[1:] + "/" + service.Name + "/"
		_, err = r.client.KV().DeleteTree(path, nil)
	}
	return err
}

func (r *ConsulMetaAdapter) Refresh(service *bridge.Service) error {
	return nil
}

func (r *ConsulMetaAdapter) Services() ([]*bridge.Service, error) {
	services, err := r.client.Agent().Services()
	if err != nil {
		return []*bridge.Service{}, err
	}
	out := make([]*bridge.Service, len(services))
	i := 0
	for _, v := range services {
		s := &bridge.Service{
			ID:   v.ID,
			Name: v.Service,
			Port: v.Port,
			Tags: v.Tags,
			IP:   v.Address,
		}
		out[i] = s
		i++
	}
	return out, nil
}
