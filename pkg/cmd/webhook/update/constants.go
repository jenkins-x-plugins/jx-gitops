package update

const (
	// LighthouseHMACToken names of the hmac token secret for Lighthouse
	LighthouseHMACToken = "lighthouse-hmac-token"

	// WebHookAnnotation annotation to indicate if the webhook has been created or failed
	WebHookAnnotation = "webhook.jenkins-x.io"

	// WebHookErrorAnnotation indicates an error to create a webhook
	WebHookErrorAnnotation = "webhook.jenkins-x.io/error"
)
