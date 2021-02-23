package v1alpha1

const (
	// APIVersion the api version
	APIVersion = "gitops.jenkins-x.io/v1alpha1"

	// KindSecretMapping the kind
	KindSecretMapping = "SecretMapping"

	// KindSourceConfig the kind
	KindSourceConfig = "SourceConfig"

	// DomainPlaceholder what is the default domain value used as a place holder until
	// the real domain name can be discovered which is usually after the first apply
	// of kubernetes resources as we need to discover the LoadBalancer Service in the nginx namespace
	DomainPlaceholder = "change.me"
)
