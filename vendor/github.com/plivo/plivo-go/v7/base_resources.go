package plivo

// BaseResource and BaseResourceInterface only supports PhloClient type as of now.
// Todo: Change client type to *BaseClient and try to pass *PhloClient to client?

type BaseResource struct {
	client *PhloClient
}

type BaseResourceInterface struct {
	client       *PhloClient
	resourceType BaseResource // Todo: Need this?
}
