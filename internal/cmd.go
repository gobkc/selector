package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// ReadInput reads strings from standard input (stdin).
// If input is piped, it reads until EOF.
// If input is interactive (from terminal), it prompts the user to enter options until EOF (Ctrl+D).
func ReadInput(options string) ([]string, error) {
	var lines []string
	reader := bufio.NewReader(os.Stdin)

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		for {
			input, err := reader.ReadString('\n')
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				return nil, fmt.Errorf("error reading input: %w", err)
			}
			lines = append(lines, input)
		}
		return lines, nil
	}
	if len(lines) == 0 {
		bs := bytes.Split([]byte(options), []byte{10})
		for _, s := range bs {
			lines = append(lines, string(s))
		}
		return lines, nil
	}
	return lines, nil
}

// BindCmd parse the command parameters and bind results to input struct
func BindCmd(params any) error {
	if reflect.TypeOf(params).Kind() != reflect.Pointer {
		return fmt.Errorf("param must be a pointer")
	}
	val := reflect.ValueOf(params).Elem()
	typ := val.Type()

	// create a map for storing fields
	fieldMap := make(map[string]reflect.Value)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldMap[strings.ToLower(field.Name)] = val.Field(i)
		short := strings.ToLower(field.Name[0:1])
		if _, ok := fieldMap[short]; !ok {
			fieldMap[short] = val.Field(i)
		}
	}

	args := os.Args[1:]

	// parse command parameters
	for _, arg := range args {
		arg = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(arg, `-`, ``)))
		var parameter string
		if strings.Contains(arg, `=`) {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				continue
			}
			arg, parameter = parts[0], parts[1]
		}
		if field, ok := fieldMap[arg]; ok {
			switch field.Kind() {
			case reflect.Bool:
				field.SetBool(true)
			case reflect.String:
				field.SetString(strings.TrimSpace(parameter))
			}
		}
	}

	return nil
}
