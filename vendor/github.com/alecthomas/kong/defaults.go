package kong

// ApplyDefaults if they are not already set.
func ApplyDefaults(target interface{}, options ...Option) error {
	app, err := New(target, options...)
	if err != nil {
		return err
	}
	ctx, err := Trace(app, nil)
	if err != nil {
		return err
	}
	err = ctx.Resolve()
	if err != nil {
		return err
	}
	if err = ctx.ApplyDefaults(); err != nil {
		return err
	}
	return ctx.Validate()
}
