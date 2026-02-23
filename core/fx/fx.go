package fx

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"strings"
)

// 定义一个简单结构体
type User struct {
	ID     int               `json:"id"`
	Name   string            `json:"name"`
	Tags   []string          `json:"tags,omitempty"`
	MapTmp map[string]string `json:"map"`
}

func Fx() {
	// ===== 基本用法 =====
	u := User{ID: 1, Name: "Vincent"}

	// v2 Marshal
	data, err := jsonv2.Marshal(u)
	if err != nil {
		panic(err)
	}
	fmt.Println("Marshal 输出:", string(data))
	// ===== 基本用法 =====
	u = User{ID: 1, Name: "Vincent", Tags: []string{"golang", "music"}}

	// v2 Marshal
	data, err = jsonv2.Marshal(u)
	if err != nil {
		panic(err)
	}
	fmt.Println("Marshal 输出:", string(data))

	// v2 Unmarshal
	var u2 User
	if err := jsonv2.Unmarshal(data, &u2); err != nil {
		panic(err)
	}
	fmt.Println("Unmarshal 结果:", u2)

	// ===== 流式解码用法 =====
	jsonArray := `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`
	dec := jsontext.NewDecoder(strings.NewReader(jsonArray))

	// 读取开头的 '['
	tok, _ := dec.ReadToken()
	if tok.Kind() != '[' {
		panic("不是数组")
	}

	fmt.Println("流式解码:")
	for dec.PeekKind() != ']' {
		var u3 User
		if err := jsonv2.UnmarshalDecode(dec, &u3); err != nil {
			panic(err)
		}
		fmt.Printf("  -> %+v\n", u3)
	}
}
