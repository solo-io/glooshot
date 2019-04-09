package generate

type Config struct {
	Namespace *Namespace        `json:"namespace,omitempty"`
	Rbac      *Rbac             `json:"rbac,omitempty"`
	Crds      *Crds             `json:"crds,omitempty"`
	ApiServer *MeshAppApiserver `json:"apiserver,omitempty"`
	Operator  *MeshAppOperator  `json:"operator,omitempty"`
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

type MeshAppApiserver struct {
	Deployment *MeshAppApiserverDeployment `json:"deployment,omitempty"`
	GrpcPort   int                         `json:"grpcPort,omitempty"`
}

type MeshAppApiserverDeployment struct {
	Image *Image `json:"image,omitempty"`
	*DeploymentSpec
}

type MeshAppOperator struct {
	Deployment *MeshAppOperatorDeployment `json:"deployment,omitempty"`
}

type MeshAppOperatorDeployment struct {
	Image *Image `json:"image,omitempty"`
	*DeploymentSpec
}
