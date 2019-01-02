package Utils

import "encoding/json"

// ToJSONString serializes a provided object to a JSON string
func ToJSONString(obj interface{}) string {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return ""
	}

	return string(bytes)
}

// FromJSONString deserializes a provided JSON string to an object
func FromJSONString(str string, out interface{}) {
	json.Unmarshal([]byte(str), out)
}
