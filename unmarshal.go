package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
)

func UnmarshalRecord(journal *sdjournal.Journal, to *Record) error {
	entry, err := journal.GetEntry()
	if err != nil {
		return err
	}
	to.TimeUsec = int64(entry.RealtimeTimestamp)
	if err := unmarshalRecord(entry, reflect.ValueOf(to).Elem()); err != nil {
		return err
	}
	i := 0
	for strings.HasPrefix(to.Message, `{"`) && !strings.HasSuffix(to.Message, `}`) {
		// the journal splits up messages of length >2K. Let's try and join them back again
		// ..for up to 10 records
		seeked, err := journal.Next()
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if seeked == 0 {
			journal.Wait(2 * time.Second)
			continue
		}
		entry, err := journal.GetEntry()
		to.Message += entry.Fields["MESSAGE"]
		i++
		if i > 10 {
			break
		}
	}
	return nil
}

func unmarshalRecord(entry *sdjournal.JournalEntry, toVal reflect.Value) error {
	toType := toVal.Type()

	numField := toVal.NumField()

	// This intentionally supports only the few types we actually
	// use on the Record struct. It's not intended to be generic.

	for i := 0; i < numField; i++ {
		fieldVal := toVal.Field(i)
		fieldDef := toType.Field(i)
		fieldType := fieldDef.Type
		fieldTag := fieldDef.Tag
		fieldTypeKind := fieldType.Kind()

		if fieldTypeKind == reflect.Struct {
			// Recursively unmarshal from the same journal
			unmarshalRecord(entry, fieldVal)
		}

		jdKey := fieldTag.Get("journald")
		if jdKey == "" {
			continue
		}

		value, ok := entry.Fields[jdKey]
		if !ok {
			fieldVal.Set(reflect.Zero(fieldType))
			continue
		}

		switch fieldTypeKind {
		case reflect.Int:
			intVal, err := strconv.Atoi(value)
			if err != nil {
				// Should never happen, but not much we can do here.
				fieldVal.Set(reflect.Zero(fieldType))
				continue
			}
			fieldVal.SetInt(int64(intVal))
			break
		case reflect.String:
			fieldVal.SetString(value)
			break
		default:
			// Should never happen
			panic(fmt.Errorf("Can't unmarshal to %s", fieldType))
		}
	}

	return nil
}
