package kong

import (
	"fmt"
	"reflect"
	"strings"
)

// Plugins are dynamically embedded command-line structures.
//
// Each element in the Plugins list *must* be a pointer to a structure.
type Plugins []interface{}

func build(k *Kong, ast interface{}) (app *Application, err error) {
	v := reflect.ValueOf(ast)
	iv := reflect.Indirect(v)
	if v.Kind() != reflect.Ptr || iv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a pointer to a struct but got %T", ast)
	}

	app = &Application{}
	extraFlags := k.extraFlags()
	seenFlags := map[string]bool{}
	for _, flag := range extraFlags {
		seenFlags[flag.Name] = true
	}

	node, err := buildNode(k, iv, ApplicationNode, newEmptyTag(), seenFlags)
	if err != nil {
		return nil, err
	}
	if len(node.Positional) > 0 && len(node.Children) > 0 {
		return nil, fmt.Errorf("can't mix positional arguments and branching arguments on %T", ast)
	}
	app.Node = node
	app.Node.Flags = append(extraFlags, app.Node.Flags...)
	app.Tag = newEmptyTag()
	app.Tag.Vars = k.vars
	return app, nil
}

func dashedString(s string) string {
	return strings.Join(camelCase(s), "-")
}

type flattenedField struct {
	field reflect.StructField
	value reflect.Value
	tag   *Tag
}

func flattenedFields(v reflect.Value, ptag *Tag) (out []flattenedField, err error) {
	v = reflect.Indirect(v)
	for i := 0; i < v.NumField(); i++ {
		ft := v.Type().Field(i)
		fv := v.Field(i)
		tag, err := parseTag(v, ft)
		if err != nil {
			return nil, err
		}
		if tag.Ignored {
			continue
		}
		// Assign group if it's not already set.
		if tag.Group == "" {
			tag.Group = ptag.Group
		}
		// Accumulate prefixes.
		tag.Prefix = ptag.Prefix + tag.Prefix
		tag.EnvPrefix = ptag.EnvPrefix + tag.EnvPrefix
		// Combine parent vars.
		tag.Vars = ptag.Vars.CloneWith(tag.Vars)
		// Command and embedded structs can be pointers, so we hydrate them now.
		if (tag.Cmd || tag.Embed) && ft.Type.Kind() == reflect.Ptr {
			fv = reflect.New(ft.Type.Elem()).Elem()
			v.FieldByIndex(ft.Index).Set(fv.Addr())
		}
		if !ft.Anonymous && !tag.Embed {
			if fv.CanSet() {
				field := flattenedField{field: ft, value: fv, tag: tag}
				out = append(out, field)
			}
			continue
		}

		// Embedded type.
		if fv.Kind() == reflect.Interface {
			fv = fv.Elem()
		} else if fv.Type() == reflect.TypeOf(Plugins{}) {
			for i := 0; i < fv.Len(); i++ {
				fields, ferr := flattenedFields(fv.Index(i).Elem(), tag)
				if ferr != nil {
					return nil, ferr
				}
				out = append(out, fields...)
			}
			continue
		}
		sub, err := flattenedFields(fv, tag)
		if err != nil {
			return nil, err
		}
		out = append(out, sub...)
	}
	return out, nil
}

// Build a Node in the Kong data model.
//
// "v" is the value to create the node from, "typ" is the output Node type.
func buildNode(k *Kong, v reflect.Value, typ NodeType, tag *Tag, seenFlags map[string]bool) (*Node, error) {
	node := &Node{
		Type:   typ,
		Target: v,
		Tag:    tag,
	}
	fields, err := flattenedFields(v, tag)
	if err != nil {
		return nil, err
	}

MAIN:
	for _, field := range fields {
		for _, r := range k.ignoreFields {
			if r.MatchString(v.Type().Name() + "." + field.field.Name) {
				continue MAIN
			}
		}

		ft := field.field
		fv := field.value

		tag := field.tag
		name := tag.Name
		if name == "" {
			name = tag.Prefix + k.flagNamer(ft.Name)
		} else {
			name = tag.Prefix + name
		}

		if len(tag.Envs) != 0 {
			for i := range tag.Envs {
				tag.Envs[i] = tag.EnvPrefix + tag.Envs[i]
			}
		}

		// Nested structs are either commands or args, unless they implement the Mapper interface.
		if field.value.Kind() == reflect.Struct && (tag.Cmd || tag.Arg) && k.registry.ForValue(fv) == nil {
			typ := CommandNode
			if tag.Arg {
				typ = ArgumentNode
			}
			err = buildChild(k, node, typ, v, ft, fv, tag, name, seenFlags)
		} else {
			err = buildField(k, node, v, ft, fv, tag, name, seenFlags)
		}
		if err != nil {
			return nil, err
		}
	}

	// Validate if there are no duplicate names
	if err := checkDuplicateNames(node, v); err != nil {
		return nil, err
	}

	// "Unsee" flags.
	for _, flag := range node.Flags {
		delete(seenFlags, "--"+flag.Name)
		if flag.Short != 0 {
			delete(seenFlags, "-"+string(flag.Short))
		}
	}

	if err := validatePositionalArguments(node); err != nil {
		return nil, err
	}

	return node, nil
}

func validatePositionalArguments(node *Node) error {
	var last *Value
	for i, curr := range node.Positional {
		if last != nil {
			// Scan through argument positionals to ensure optional is never before a required.
			if !last.Required && curr.Required {
				return fmt.Errorf("%s: required %q cannot come after optional %q", node.FullPath(), curr.Name, last.Name)
			}

			// Cumulative argument needs to be last.
			if last.IsCumulative() {
				return fmt.Errorf("%s: argument %q cannot come after cumulative %q", node.FullPath(), curr.Name, last.Name)
			}
		}

		last = curr
		curr.Position = i
	}

	return nil
}

func buildChild(k *Kong, node *Node, typ NodeType, v reflect.Value, ft reflect.StructField, fv reflect.Value, tag *Tag, name string, seenFlags map[string]bool) error {
	child, err := buildNode(k, fv, typ, newEmptyTag(), seenFlags)
	if err != nil {
		return err
	}
	child.Name = name
	child.Tag = tag
	child.Parent = node
	child.Help = tag.Help
	child.Hidden = tag.Hidden
	child.Group = buildGroupForKey(k, tag.Group)
	child.Aliases = tag.Aliases

	if provider, ok := fv.Addr().Interface().(HelpProvider); ok {
		child.Detail = provider.Help()
	}

	// A branching argument. This is a bit hairy, as we let buildNode() do the parsing, then check that
	// a positional argument is provided to the child, and move it to the branching argument field.
	if tag.Arg {
		if len(child.Positional) == 0 {
			return failField(v, ft, "positional branch must have at least one child positional argument named %q", name)
		}
		if child.Positional[0].Name != name {
			return failField(v, ft, "first field in positional branch must have the same name as the parent field (%s).", child.Name)
		}

		child.Argument = child.Positional[0]
		child.Positional = child.Positional[1:]
		if child.Help == "" {
			child.Help = child.Argument.Help
		}
	} else {
		if tag.HasDefault {
			if node.DefaultCmd != nil {
				return failField(v, ft, "can't have more than one default command under %s", node.Summary())
			}
			if tag.Default != "withargs" && (len(child.Children) > 0 || len(child.Positional) > 0) {
				return failField(v, ft, "default command %s must not have subcommands or arguments", child.Summary())
			}
			node.DefaultCmd = child
		}
		if tag.Passthrough {
			if len(child.Children) > 0 || len(child.Flags) > 0 {
				return failField(v, ft, "passthrough command %s must not have subcommands or flags", child.Summary())
			}
			if len(child.Positional) != 1 {
				return failField(v, ft, "passthrough command %s must contain exactly one positional argument", child.Summary())
			}
			if !checkPassthroughArg(child.Positional[0].Target) {
				return failField(v, ft, "passthrough command %s must contain exactly one positional argument of []string type", child.Summary())
			}
			child.Passthrough = true
		}
	}
	node.Children = append(node.Children, child)

	if len(child.Positional) > 0 && len(child.Children) > 0 {
		return failField(v, ft, "can't mix positional arguments and branching arguments")
	}

	return nil
}

func buildField(k *Kong, node *Node, v reflect.Value, ft reflect.StructField, fv reflect.Value, tag *Tag, name string, seenFlags map[string]bool) error {
	mapper := k.registry.ForNamedValue(tag.Type, fv)
	if mapper == nil {
		return failField(v, ft, "unsupported field type %s, perhaps missing a cmd:\"\" tag?", ft.Type)
	}

	value := &Value{
		Name:         name,
		Help:         tag.Help,
		OrigHelp:     tag.Help,
		HasDefault:   tag.HasDefault,
		Default:      tag.Default,
		DefaultValue: reflect.New(fv.Type()).Elem(),
		Mapper:       mapper,
		Tag:          tag,
		Target:       fv,
		Enum:         tag.Enum,
		Passthrough:  tag.Passthrough,

		// Flags are optional by default, and args are required by default.
		Required: (!tag.Arg && tag.Required) || (tag.Arg && !tag.Optional),
		Format:   tag.Format,
	}

	if tag.Arg {
		node.Positional = append(node.Positional, value)
	} else {
		if seenFlags["--"+value.Name] {
			return failField(v, ft, "duplicate flag --%s", value.Name)
		}
		seenFlags["--"+value.Name] = true
		if tag.Short != 0 {
			if seenFlags["-"+string(tag.Short)] {
				return failField(v, ft, "duplicate short flag -%c", tag.Short)
			}
			seenFlags["-"+string(tag.Short)] = true
		}
		flag := &Flag{
			Value:       value,
			Short:       tag.Short,
			PlaceHolder: tag.PlaceHolder,
			Envs:        tag.Envs,
			Group:       buildGroupForKey(k, tag.Group),
			Xor:         tag.Xor,
			Hidden:      tag.Hidden,
		}
		value.Flag = flag
		node.Flags = append(node.Flags, flag)
	}
	return nil
}

func buildGroupForKey(k *Kong, key string) *Group {
	if key == "" {
		return nil
	}
	for _, group := range k.groups {
		if group.Key == key {
			return &group
		}
	}

	// No group provided with kong.ExplicitGroups. We create one ad-hoc for this key.
	return &Group{
		Key:   key,
		Title: key,
	}
}

func checkDuplicateNames(node *Node, v reflect.Value) error {
	seenNames := make(map[string]struct{})
	for _, node := range node.Children {
		if _, ok := seenNames[node.Name]; ok {
			name := v.Type().Name()
			if name == "" {
				name = "<anonymous struct>"
			}
			return fmt.Errorf("duplicate command name %q in command %q", node.Name, name)
		}

		seenNames[node.Name] = struct{}{}
	}

	return nil
}
