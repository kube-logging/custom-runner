package config

type ArrayMode string

const (
	ImapArrayIdAppend     ArrayMode = "[+]"
	ImapArrayIdUpdateLast ArrayMode = "[-]"
)

type ISet interface {
	GetIn(keys ...interface{}) interface{}
	SetIn(keys []string, value string) interface{}
}

type Imap map[interface{}]interface{}

func (s Imap) GetIn(keys ...interface{}) interface{} {
	if len(keys) == 0 {
		return s
	}
	key, subKeys := keys[0], keys[1:]
	v, ok := key.(string)
	if !ok {
		return nil
	}
	subStore, ok := s[v]
	if !ok {
		return nil
	}
	switch v := subStore.(type) {
	case Imap:
		return v.GetIn(subKeys...)
	case []interface{}:
		return ImapArray(v).GetIn(subKeys...)
	default:
		if len(subKeys) > 0 {
			return nil
		}
		return v
	}
}

type ImapArray []interface{}

func (s ImapArray) GetIn(keys ...interface{}) interface{} {
	if len(keys) == 0 {
		return []interface{}(s)
	}
	key, subKeys := keys[0], keys[1:]
	intKey, ok := key.(int)
	if !ok {
		return nil
	}

	if len(s) <= intKey || intKey < 0 {
		return nil
	}
	subStore := s[intKey]
	switch v := subStore.(type) {
	case Imap:
		return v.GetIn(subKeys...)
	case []interface{}:
		return ImapArray(v).GetIn(subKeys...)
	default:
		if len(subKeys) > 0 {
			return nil
		}
		return v
	}

}

func getSubImap(s ISet, key interface{}) Imap {
	var x interface{}
	switch t := s.(type) {
	case ImapArray:
		if key == string(ImapArrayIdUpdateLast) && len(t) > 0 {
			x = t[len(t)-1]
		} else {
			x = nil
		}
	default:
		x = t.GetIn(key)
	}

	if v, ok := x.(Imap); ok {
		return v
	}
	return Imap{}
}

func getSubImapArray(s ISet, key interface{}) ImapArray {
	var x interface{}
	switch t := s.(type) {
	case ImapArray:
		if key == string(ImapArrayIdUpdateLast) && len(t) > 0 {
			x = t[len(t)-1]
		} else {
			x = nil
		}
	default:
		x = t.GetIn(key)
	}

	if v, ok := x.(ImapArray); ok {
		return v
	}
	return ImapArray{}
}

func (s Imap) SetIn(keys []string, value string) interface{} {
	if len(keys) == 0 {
		return s
	}
	key, subKeys := keys[0], keys[1:]
	if len(subKeys) == 0 {
		s[key] = value
		return s
	}

	switch subKeys[0] {
	case string(ImapArrayIdAppend), string(ImapArrayIdUpdateLast):
		subArray := getSubImapArray(s, key)
		subArray = subArray.SetIn(subKeys, value).(ImapArray)
		s[key] = subArray
	default:
		subMap := getSubImap(s, key)
		subMap.SetIn(subKeys, value)
		s[key] = subMap
	}

	return s
}

func (s ImapArray) update(arrayMode ArrayMode, value interface{}) ImapArray {
	switch arrayMode {
	case ImapArrayIdUpdateLast:
		if len(s) == 0 {
			s = append(s, value)
		} else {
			s[len(s)-1] = value
		}
	default:
		s = append(s, value)
	}
	return s
}

func (s ImapArray) SetIn(keys []string, value string) interface{} {
	if len(keys) == 0 {
		return s
	}
	key, subKeys := keys[0], keys[1:]
	if len(subKeys) == 0 {
		s = s.update(ArrayMode(key), value)
		return s
	}

	switch subKeys[0] {
	case string(ImapArrayIdAppend), string(ImapArrayIdUpdateLast):
		subArray := getSubImapArray(s, key)
		subArray = subArray.SetIn(subKeys, value).(ImapArray)
		s = s.update(ArrayMode(key), subArray)
		// s = append(s, subArray)
	default:
		subMap := getSubImap(s, key)
		subMap.SetIn(subKeys, value)
		s = s.update(ArrayMode(key), subMap)
		// s = append(s, subMap)
	}

	return s
}
