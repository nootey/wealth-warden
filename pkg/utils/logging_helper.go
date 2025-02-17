package utils

type Changes struct {
	New map[string]string
	Old map[string]string
}

func InitChanges() *Changes {
	return &Changes{
		New: make(map[string]string),
		Old: make(map[string]string),
	}
}

func CompareChanges(old, new string, obj *Changes, index string) {
	if old != new {
		if len(new) == 0 {
			obj.Old[index] = old
		} else if len(old) == 0 {
			obj.New[index] = new
		} else {
			obj.Old[index] = old
			obj.New[index] = new
		}
	}
}
