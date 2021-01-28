// Shows a sample usage of the ETW logging package.
package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/Microsoft/go-winio/pkg/etw"
	"github.com/Microsoft/go-winio/pkg/guid"
	"github.com/sirupsen/logrus"
)

func callback(sourceID guid.GUID, state etw.ProviderState, level etw.Level, matchAnyKeyword uint64, matchAllKeyword uint64, filterData uintptr) {
	fmt.Printf("Callback: isEnabled=%d, level=%d, matchAnyKeyword=%d\n", state, level, matchAnyKeyword)
}

func main() {
	provider, err := etw.NewProvider("TestProvider", callback)

	if err != nil {
		logrus.Error(err)
		return
	}
	defer func() {
		if err := provider.Close(); err != nil {
			logrus.Error(err)
		}
	}()

	fmt.Printf("Provider ID: %s\n", provider)

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Press enter to log events")
	reader.ReadString('\n')

	if err := provider.WriteEvent(
		"TestEvent",
		etw.WithEventOpts(
			etw.WithLevel(etw.LevelInfo),
			etw.WithKeyword(0x140),
		),
		etw.WithFields(
			etw.StringField("TestField", "Foo"),
			etw.StringField("TestField2", "Bar"),
			etw.Struct("TestStruct",
				etw.StringField("Field1", "Value1"),
				etw.StringField("Field2", "Value2")),
			etw.StringArray("TestArray", []string{
				"Item1",
				"Item2",
				"Item3",
				"Item4",
				"Item5",
			})),
	); err != nil {
		logrus.Error(err)
		return
	}
}
