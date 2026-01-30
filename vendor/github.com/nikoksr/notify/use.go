package notify

// useService adds a given service to the Notifier's services list.
func (n *Notify) useService(service Notifier) {
	if service != nil {
		n.notifiers = append(n.notifiers, service)
	}
}

// useServices adds the given service(s) to the Notifier's services list.
func (n *Notify) useServices(services ...Notifier) {
	for _, s := range services {
		n.useService(s)
	}
}

// UseServices adds the given service(s) to the Notifier's services list.
func (n *Notify) UseServices(services ...Notifier) {
	n.useServices(services...)
}

// UseServices adds the given service(s) to the Notifier's services list.
func UseServices(services ...Notifier) {
	std.UseServices(services...)
}
