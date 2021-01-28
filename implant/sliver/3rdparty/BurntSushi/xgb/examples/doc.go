/*
Package examples contains a few different use cases of XGB, like creating
a window, reading properties, and querying for information about multiple
heads using the Xinerama or RandR extensions.

If you're looking to get started quickly, I recommend checking out the
create-window example first. It is the most documented and probably covers
some of the more common bare bones cases of creating windows and responding
to events.

If you're looking to query information about your window manager,
get-active-window is a start. However, to do anything extensive requires
a lot of boiler plate. To that end, I'd recommend use of my higher level
library, xgbutil: https://github.com/BurntSushi/xgbutil

There are also examples of using the Xinerama and RandR extensions, if you're
interested in querying information about your active heads. In RandR's case,
you can also reconfigure your heads, but the example doesn't cover that.

*/
package documentation
