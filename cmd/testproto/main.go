package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/manniwood/iidy"
)

func main() {
	li := &iidy.Entry{
		List: "test list",
		Item: "test value",
	}
	// Write the list item
	out, err := proto.Marshal(li)
	if err != nil {
		log.Fatalln("Failed to encode list item:", err)
	}
	if err := ioutil.WriteFile("list-item.proto", out, 0644); err != nil {
		log.Fatalln("Failed to write file:", err)
	}
	// Read the list item back in
	in, err := ioutil.ReadFile("list-item.proto")
	if err != nil {
		log.Fatalln("Error reading file:", err)
	}
	readLi := &iidy.Entry{}
	if err := proto.Unmarshal(in, readLi); err != nil {
		log.Fatalln("Failed to parse address book:", err)
	}
	fmt.Printf("The read-in list item is %v\n", readLi)
}
