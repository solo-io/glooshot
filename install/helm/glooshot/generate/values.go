package generate

type Config struct {
	Namespace *Namespace `json:"namespace,omitempty"`
	Rbac      *Rbac      `json:"rbac,omitempty"`
	Crds      *Crds      `json:"crds,omitempty"`
	Glooshot  *Glooshot  `json:"glooshot,omitempty"`
}

type Namespace struct {
	Create bool `json:"create"`
}

type Rbac struct {
	Create bool `json:"create"`
}

type Crds struct {
	Create bool `json:"create"`
}

// Common
type Image struct {
	Tag        string `json:"tag"`
	Repository string `json:"repository"`
	PullPolicy string `json:"pullPolicy"`
	PullSecret string `json:"pullSecret,omitempty"`
}

type DeploymentSpec struct {
	Replicas int  `json:"replicas"`
	Stats    bool `json:"stats"`
}

type Settings struct {
	WatchNamespaces []string    `json:"watchNamespaces,omitempty"`
	WriteNamespace  string      `json:"writeNamespace,omitempty"`
	Extensions      interface{} `json:"extensions,omitempty"`
}

type Glooshot struct {
	Deployment *GlooshotDeployment `json:"deployment,omitempty"`
}

type GlooshotDeployment struct {
	Image *Image `json:"image,omitempty"`
	*DeploymentSpec
}
