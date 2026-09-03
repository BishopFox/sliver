package opfor

// The objects returned by ScriptLoader.getScripts() and getScriptsByKey() are
// the loader's actual registries in Sleep 2.1, not snapshots. Keeping the
// portable Java collection objects authoritative preserves both reference
// identity and intentionally independent mutation: changing one registry does
// not silently repair the other.

func (loader *portableScriptLoader) firstLoadedScript() (Value, bool) {
	if loader == nil || loader.loadedScripts == nil {
		return Null(), false
	}
	value, err := loader.loadedScripts.getAt(0)
	return value, err == nil
}

func (loader *portableScriptLoader) addLoadedScript(instance *portableScriptInstance) {
	if loader == nil || loader.loadedScripts == nil || instance == nil {
		return
	}
	collection := loader.loadedScripts
	collection.mu.Lock()
	collection.values = append(collection.values, ObjectValue(instance))
	collection.mod++
	collection.mu.Unlock()
}

func (loader *portableScriptLoader) removeLoadedScript(instance *portableScriptInstance) {
	if loader == nil || loader.loadedScripts == nil || instance == nil {
		return
	}
	collection := loader.loadedScripts
	needle := ObjectValue(instance)
	collection.mu.Lock()
	for index, candidate := range collection.values {
		if !candidate.IdentityEqual(needle) {
			continue
		}
		copy(collection.values[index:], collection.values[index+1:])
		collection.values[len(collection.values)-1] = Null()
		collection.values = collection.values[:len(collection.values)-1]
		collection.mod++
		break
	}
	collection.mu.Unlock()
}

func (loader *portableScriptLoader) scriptByKey(key Value) (Value, bool) {
	if loader == nil || loader.scriptsByKey == nil {
		return Null(), false
	}
	mapping := loader.scriptsByKey
	text := sleepCanonicalString(key)
	mapping.mu.RLock()
	value, present := mapping.values[text]
	mapping.mu.RUnlock()
	return value, present
}

func (loader *portableScriptLoader) putScriptByKey(key, value Value) Value {
	if loader == nil || loader.scriptsByKey == nil {
		return Null()
	}
	mapping := loader.scriptsByKey
	text, keyValue := sleepHashKey(key)
	mapping.mu.Lock()
	previous, present := mapping.values[text]
	if !present {
		mapping.keys = append(mapping.keys, text)
		mapping.keyValues[text] = keyValue
		mapping.entries[text] = &portableJavaMapEntry{
			mapping: mapping, key: text, keyValue: keyValue, value: value,
		}
		mapping.mod++
	} else if entry := mapping.entries[text]; entry != nil {
		entry.mu.Lock()
		entry.value = value
		entry.mu.Unlock()
	}
	mapping.values[text] = value
	mapping.mu.Unlock()
	if !present {
		return Null()
	}
	return previous
}

func (loader *portableScriptLoader) removeScriptByKey(key Value) Value {
	if loader == nil || loader.scriptsByKey == nil {
		return Null()
	}
	mapping := loader.scriptsByKey
	text := sleepCanonicalString(key)
	mapping.mu.Lock()
	value, present := mapping.removeKeyLocked(text)
	mapping.mu.Unlock()
	if !present {
		return Null()
	}
	return value
}

func (loader *portableScriptLoader) scriptKeys() []Value {
	if loader == nil || loader.scriptsByKey == nil {
		return nil
	}
	mapping := loader.scriptsByKey
	mapping.mu.RLock()
	keys := mapping.wrapperKeysLocked()
	values := make([]Value, 0, len(keys))
	for _, key := range keys {
		if _, present := mapping.values[key]; present {
			values = append(values, mapping.keyValues[key])
		}
	}
	mapping.mu.RUnlock()
	return values
}

func (loader *portableScriptLoader) clearRegistries() {
	if loader == nil {
		return
	}
	if loader.loadedScripts != nil {
		_ = loader.loadedScripts.clearValues()
	}
	if loader.scriptsByKey != nil {
		_, _, _ = loader.scriptsByKey.invoke(ObjectInvocation{Op: ObjectInvoke, Message: "clear"})
	}
}
