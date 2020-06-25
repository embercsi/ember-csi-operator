package embercsi

import (
	"testing"
)

func TestMarshal(t *testing.T) {
	// YAML map
	input_map := make(map[string]interface{})
	input_map["key"] = "val"
	expected := "{\"key\":\"val\"}"
	result, err := interfaceToString(input_map)
	if result != expected {
		t.Errorf("Failed to marshal valid YAML map, got %v\n", result)
	}
	if err != nil {
		t.Errorf("Error return is wrong, got %v\n",  err)
	}


	// JSON string
	input_string := "{\"key\":\"val\"}"
	result, err = interfaceToString(input_string)
	if result != expected {
		t.Errorf("Failed to marshal valid JSON string, got %v\n", result)
	}
	if err != nil {
		t.Errorf("Error return is wrong, got %v\n",  err)
	}


	// Invalid JSON string
	input_string = "{key\":\"val\"}"
	expected = "{key\":\"val\"}"
	result, err = interfaceToString(input_string)
	if result != expected {
		t.Errorf("Failed to marshal invalid JSON string, got %v\n", result)
	}
	if err == nil {
		t.Errorf("Error return is wrong, got %v\n",  err)
	}


	// Invalid data type
	result, err = interfaceToString(42)
	if result != "{}" {
		t.Errorf("Failed to marshal invalid JSON string, got %v\n", result)
	}
	if err == nil {
		t.Errorf("Error return is wrong, got %v\n",  err)
	}


	// JSON with escape chars
	input_string = "{\"key\":\"C:\\\\somepath\"}"
	expected = "{\"key\":\"C:\\\\somepath\"}"
	result, err = interfaceToString(input_string)
	if result != expected {
		t.Errorf("Failed to marshal invalid JSON string, got %v, expected %v\n", result, expected)
	}
	if err != nil {
		t.Errorf("Error return is wrong, got %v\n",  err)
	}

	// nil
	result, err = interfaceToString(nil)
	if result != "{}" {
		t.Errorf("Failed to marshal nil, got %v\n", result)
	}
	if err != nil {
		t.Errorf("Error return is wrong, got %v\n",  err)
	}
}

func TestConfigTransform(t *testing.T) {
	input := "{\"key__transform_empty_none\":\"\"}"
	expected := "{\"key\":null}"
	result := configTransform(input)
	if result != expected {
		t.Errorf("Failed to transform, got %v, expected %v\n", result, expected)
	}

	input = "{\"key__transform_csv\":\"a,b,c\"}"
	expected = "{\"key\":[\"a\",\"b\",\"c\"]}"
	result = configTransform(input)
	if result != expected {
		t.Errorf("Failed to transform, got %v, expected %v\n", result, expected)
	}

	input = "{\"key__transform_csv_kvs\":\"a:b\"}"
	expected = "{\"key\":{\"a\":\"b\"}}"
	result = configTransform(input)
	if result != expected {
		t.Errorf("Failed to transform, got %v, expected %v\n", result, expected)
	}

	input = "{\"driver\":\"Someone\",\"driver__Someone__option\":\"abc\",\"driver__Other__option\":\"def\"}"
	expected = "{\"driver\":\"Someone\",\"option\":\"abc\"}"
	result = configTransform(input)
	if result != expected {
		t.Errorf("Failed to transform, got %v, expected %v\n", result, expected)
	}

	input = "{\"driver\":\"Someone\",\"driver__Someone__option\":\"C:\\\\somepath\",\"driver__Other__option\":\"def\"}"
	expected = "{\"driver\":\"Someone\",\"option\":\"C:\\\\somepath\"}"
	result = configTransform(input)
	if result != expected {
		t.Errorf("Failed to transform, got %v, expected %v\n", result, expected)
	}
	input = "{\"key__transform_string_float\":\"3.14159\"}"
	expected = "{\"key\":3.14159}"
	result = configTransform(input)
	if result != expected {
		t.Errorf("Failed to transform, got %v, expected %v\n", result, expected)
	}
	input = "{\"key__transform_string_float\":\"pi\"}"
	expected = "{\"key\":\"pi\"}"
	result = configTransform(input)
	if result != expected {
		t.Errorf("Failed to transform, got %v, expected %v\n", result, expected)
	}
}


func TestSetJsonKeyIfEmpty(t *testing.T) {
	// key already set, but empty
	json := "{\"key\":\"\"}"
	expected := "{\"key\":\"42\"}"
	setJsonKeyIfEmpty(&json, "key", "42")
	if json != expected {
		t.Errorf("Failed to transform, got %v, expected %v\n", json, expected)
	}

	// key not set at all
	json = "{}"
	expected = "{\"key\":\"42\"}"
	setJsonKeyIfEmpty(&json, "key", "42")
	if json != expected {
		t.Errorf("Failed to transform, got %v, expected %v\n", json, expected)
	}

	// key already set, do not overwrite
	json = "{\"key\":\"23\"}"
	expected = "{\"key\":\"23\"}"
	setJsonKeyIfEmpty(&json, "key", "42")
	if json != expected {
		t.Errorf("Failed to transform, got %v, expected %v\n", json, expected)
	}
}
