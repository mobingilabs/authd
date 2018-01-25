package params

var (
	Port string // server port

	RunK8s bool // true if we are running in k8s

	PublicPemFile  string
	PrivatePemFile string

	Region string // aws region for resource access, valid only when not running on k8s
	Bucket string // bucket name where keyfiles are stored, valid only when not running on k8s
)
