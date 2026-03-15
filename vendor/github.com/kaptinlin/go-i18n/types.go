package i18n

// Vars holds named variables for MessageFormat interpolation in translation
// strings. Keys are placeholder names and values are substituted during
// formatting.
//
// Pass a Vars value to [Localizer.Get], [Localizer.GetX], or
// [Localizer.Format] to supply runtime data such as counts, names,
// or dates:
//
//	localizer.Get("greeting", i18n.Vars{"name": "Alice", "count": 3})
type Vars map[string]any
