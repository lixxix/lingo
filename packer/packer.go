package packer

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
)

func pack(writer *bytes.Buffer, val reflect.Value) {
	switch val.Kind() {
	case reflect.Int:
		binary.Write(writer, binary.BigEndian, val.Int())
	case reflect.Uint:
		binary.Write(writer, binary.BigEndian, val.Uint())
	case reflect.Int32:
		binary.Write(writer, binary.BigEndian, int32(val.Int()))
	case reflect.Uint32:
		binary.Write(writer, binary.BigEndian, uint32(val.Uint()))
	case reflect.Float32:
		binary.Write(writer, binary.BigEndian, float32(val.Float()))
	case reflect.Float64:
		binary.Write(writer, binary.BigEndian, val.Float())
	case reflect.Int16:
		binary.Write(writer, binary.BigEndian, int16(val.Int()))
	case reflect.Uint16:
		binary.Write(writer, binary.BigEndian, uint16(val.Uint()))
	case reflect.Int8:
		binary.Write(writer, binary.BigEndian, int8(val.Int()))
	case reflect.Uint8:
		binary.Write(writer, binary.BigEndian, uint8(val.Int()))
	case reflect.Bool:
		binary.Write(writer, binary.BigEndian, val.IsValid())
	case reflect.Int64:
		binary.Write(writer, binary.BigEndian, val.Int())
	case reflect.Uint64:
		binary.Write(writer, binary.BigEndian, uint64(val.Uint()))
	case reflect.String:
		sz := len(val.String())
		binary.Write(writer, binary.BigEndian, uint16(sz))
		binary.Write(writer, binary.BigEndian, []byte(val.String()))
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			f := val.Field(i)
			pack(writer, f)
		}
	}
}

func packData(writer *bytes.Buffer, object interface{}) {
	typ_ := reflect.ValueOf(object).Kind()
	if typ_ == reflect.Pointer {
		typeData := reflect.ValueOf(object).Elem()
		for i := 0; i < typeData.NumField(); i++ {
			f := typeData.Field(i)
			switch f.Kind() {
			case reflect.Slice:
				binary.Write(writer, binary.BigEndian, int16(f.Len()))
				for v := 0; v < f.Len(); v++ {
					pack(writer, f.Index(v))
				}
			default:
				pack(writer, f)
			}
		}
	} else {
		typeData := reflect.ValueOf(object)
		count := typeData.NumField()
		for i := 0; i < count; i++ {
			f := typeData.Field(i)
			switch f.Kind() {
			case reflect.Slice:
				binary.Write(writer, binary.BigEndian, int16(f.Len()))
				for v := 0; v < f.Len(); v++ {
					pack(writer, f.Index(v))
				}
			default:
				pack(writer, f)
			}
		}
	}
}

func PackMessage(object interface{}) []byte {
	if object == nil || reflect.ValueOf(object).IsNil() {
		return []byte{}
	}

	writer := bytes.NewBuffer([]byte{})
	packData(writer, object)
	return writer.Bytes()
}

func parser(reader *bytes.Reader, val reflect.Value) {
	switch val.Kind() {
	case reflect.Bool:
		rb := bool(false)
		binary.Read(reader, binary.BigEndian, &rb)
		val.SetBool(rb)
	case reflect.Int8:
		rb := int8(0)
		binary.Read(reader, binary.BigEndian, &rb)
		val.SetInt(int64(rb))
	case reflect.Uint8:
		rb := uint8(0)
		binary.Read(reader, binary.BigEndian, &rb)
		val.SetUint(uint64(rb))
	case reflect.Int16:
		rb := int16(0)
		binary.Read(reader, binary.BigEndian, &rb)
		val.SetInt(int64(rb))
	case reflect.Uint16:
		rb := uint16(0)
		binary.Read(reader, binary.BigEndian, &rb)
		val.SetUint(uint64(rb))
	case reflect.Int32:
		rb := int32(0)
		binary.Read(reader, binary.BigEndian, &rb)
		val.SetInt(int64(rb))
	case reflect.Uint32:
		rb := uint32(0)
		binary.Read(reader, binary.BigEndian, &rb)
		val.SetUint(uint64(rb))
	case reflect.Int64, reflect.Int:
		rb := int64(0)
		binary.Read(reader, binary.BigEndian, &rb)
		val.SetInt(rb)
	case reflect.Uint64, reflect.Uint:
		rb := uint64(0)
		binary.Read(reader, binary.BigEndian, &rb)
		val.SetUint(rb)
	case reflect.String:
		sz := int16(0)
		binary.Read(reader, binary.BigEndian, &sz)
		reader_size := make([]byte, sz)
		binary.Read(reader, binary.BigEndian, &reader_size)
		val.SetString(string(reader_size))
	case reflect.Slice:
		sz := int16(0)
		binary.Read(reader, binary.BigEndian, &sz)
		arr := reflect.MakeSlice(val.Type(), int(sz), int(sz))
		for s := int16(0); s < sz; s++ {
			parser(reader, arr.Index(int(s)))
		}
		val.Set(arr)
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			f := val.Field(i)
			parser(reader, f)
		}
	}
}

// 必须传递指针，不然无法赋值
func parserReader(reader *bytes.Reader, object interface{}) error {
	val := reflect.ValueOf(object)
	if val.Kind() == reflect.Pointer {
		elems := val.Elem()
		parser(reader, elems)
	} else {
		return errors.New("parser object must be a pointer")
	}
	return nil
}

// 传递指针
func ParserData(buff []byte, object interface{}) error {
	reader := bytes.NewReader(buff)
	return parserReader(reader, object)
}
