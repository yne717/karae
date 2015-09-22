package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/yne717/gousb/usb"
)

const MAX_SIZE = 64
const IR_FREQ = 38000
const IR_SEND_DATA_USB_SEND_MAX_LEN = 14
const C_NEED_DATA_LEN = 1224
const BUFF_SIZE = 1304

var (
	Device = flag.String("device", "22ea:0039", "select device. default \"22ea:0039\" ")
	Key    = flag.String("key", "none", "select key.")
	Number = flag.Int("number", 999999, "select number.")
	Debug  = flag.Int("debug", 3, "Debug level for libusb")
)

func main() {
	flag.Parse()

	ctx := usb.NewContext()
	// defer func() {
	// 	ctx.Close()
	// }()

	ctx.Debug(*Debug)

	devs, err := ctx.ListDevices(func(desc *usb.Descriptor) bool {
		if fmt.Sprintf("%s:%s", desc.Vendor, desc.Product) != *Device {
			return false
		}
		return true
	})
	// defer func() {
	// 	for _, dev := range devs {
	// 		dev.Close()
	// 	}
	//
	// }()

	if err != nil {
		log.Fatalf("usb.Open: %v", err)
	}

	if len(devs) == 0 {
		log.Fatal("not device.")
	}

	var ir_data []byte
	var ir_data_size int
	if *Key != "none" {
		ir_data, ir_data_size = getDataByKey(*Key)
	} else {
		ir_data, ir_data_size = getDataByNumber(*Number)
	}

	ep, err := devs[0].OpenEndpoint(uint8(1), uint8(0), uint8(0), uint8(1)|uint8(usb.ENDPOINT_DIR_OUT))
	if err != nil {
		log.Fatalf("open device faild: %s", err)
	}

	transfer(ep, ir_data, ir_data_size)

	devs[0].Close()
	ctx.Close()
}

func transfer(ep usb.Endpoint, ir_data []byte, ir_data_size int) {
	var (
		buf          []byte = make([]byte, MAX_SIZE, MAX_SIZE)
		send_bit_num int    = 0
		send_bit_pos int    = 0
		set_bit_size int    = 0
		fi           int    = 0
	)

	send_bit_num = ir_data_size / 4

	for {
		buf = make([]byte, MAX_SIZE, MAX_SIZE)
		for i, _ := range buf {
			buf[i] = 0xFF
		}

		buf[0] = 0x34
		buf[1] = byte((send_bit_num >> 8) & 0xFF)
		buf[2] = byte(send_bit_num & 0xFF)
		buf[3] = byte((send_bit_pos >> 8) & 0xFF)
		buf[4] = byte(send_bit_pos & 0xFF)

		if send_bit_num > send_bit_pos {
			set_bit_size = send_bit_num - send_bit_pos
			if set_bit_size > IR_SEND_DATA_USB_SEND_MAX_LEN {
				set_bit_size = IR_SEND_DATA_USB_SEND_MAX_LEN
			}
		} else {
			set_bit_size = 0
		}

		buf[5] = byte(set_bit_size & 0xFF)

		if set_bit_size > 0 {
			for fi = 0; fi < set_bit_size; fi++ {
				buf[6+(fi*4)] = ir_data[send_bit_pos*4]
				buf[6+(fi*4)+1] = ir_data[(send_bit_pos*4)+1]
				buf[6+(fi*4)+2] = ir_data[(send_bit_pos*4)+2]
				buf[6+(fi*4)+3] = ir_data[(send_bit_pos*4)+3]
				send_bit_pos++
			}

			_, err := ep.Write(buf)
			if err != nil {
				log.Fatalf("control faild: %v", err)
			}
		} else {
			break
		}
	}

	buf = make([]byte, MAX_SIZE, MAX_SIZE)
	for i, _ := range buf {
		buf[i] = 0xFF
	}

	buf[0] = 0x35
	buf[1] = byte((IR_FREQ >> 8) & 0xFF)
	buf[2] = byte(IR_FREQ & 0xFF)
	buf[3] = byte((send_bit_num >> 8) & 0xFF)
	buf[4] = byte(send_bit_num & 0xFF)

	_, err := ep.Write(buf)
	if err != nil {
		log.Fatalf("control faild: %v", err)
	}

}

func getDataByKey(key string) ([]byte, int) {
	list := getKeyList()
	return list[key], len(list[key])
}

func getDataByNumber(code_no int) ([]byte, int) {
	var (
		code_size       int    = 0
		buff_set_pos    int    = 0
		set_data_count  int    = 0
		fi              int    = 0
		fj              int    = 0
		fk              int    = 0
		data_pos        int    = 0
		buff_size       int    = BUFF_SIZE
		c_need_data_len int    = C_NEED_DATA_LEN
		buff            []byte = make([]byte, buff_size, buff_size)
	)

	code := [][]byte{
		{0x81, 0x16, 0x00, 0x00},
		{0x81, 0x16, 0x00, 0x00},
		{0x81, 0x16, 0x00, 0x00},
		{0x81, 0x16, 0x00, 0x00},
		{0x81, 0x16, 0x00, 0x00},
		{0x81, 0x16, 0x00, 0x00},
		{0x81, 0x16, 0x2B, 0xD4},
		{0x81, 0x16, 0x00, 0xFF},
		{0x81, 0x16, 0x1F, 0xE0},
	}

	c_reader_code := []byte{0x01, 0x56, 0x00, 0xAB}
	c_off_code := []byte{0x00, 0x15, 0x00, 0x15}
	c_on_code := []byte{0x00, 0x15, 0x00, 0x40}
	c_end_code := []byte{0x00, 0x15, 0x0E, 0xE8}

	c_num_data := []byte{0x00, 0x07, 0x0A, 0x58, 0x03, 0x06, 0x50, 0x01, 0x0E, 0x5D}

	code[0][2] = byte(c_num_data[((code_no % 1000000) / 100000)])
	code[0][3] = byte(^code[0][2] & 0xFF)
	code[1][2] = byte(c_num_data[((code_no % 100000) / 10000)])
	code[1][3] = byte(^code[1][2] & 0xFF)
	code[2][2] = byte(c_num_data[((code_no % 10000) / 1000)])
	code[2][3] = byte(^code[2][2] & 0xFF)
	code[3][2] = byte(c_num_data[((code_no % 1000) / 100)])
	code[3][3] = byte(^code[3][2] & 0xFF)
	code[4][2] = byte(c_num_data[((code_no % 100) / 10)])
	code[4][3] = byte(^code[4][2] & 0xFF)
	code[5][2] = byte(c_num_data[(code_no % 10)])
	code[5][3] = byte(^code[5][2] & 0xFF)

	if c_need_data_len <= buff_size {
		for fi = 0; fi < len(code); fi++ {
			for data_pos = 0; data_pos < len(c_reader_code); data_pos++ {
				buff[buff_set_pos] = c_reader_code[data_pos]
				buff_set_pos++
				set_data_count++
			}
			for fj = 0; fj < len(code[0]); fj++ {
				for fk = 0; fk < 8; fk++ {
					if ((code[fi][fj] >> byte(fk)) & 0x01) == 1 {
						for data_pos = 0; data_pos < len(c_on_code); data_pos++ {
							buff[buff_set_pos] = c_on_code[data_pos]
							buff_set_pos++
							set_data_count++
						}
					} else {
						for data_pos = 0; data_pos < len(c_off_code); data_pos++ {
							buff[buff_set_pos] = c_off_code[data_pos]
							buff_set_pos++
							set_data_count++
						}
					}
				}
			}
			for data_pos = 0; data_pos < len(c_end_code); data_pos++ {
				buff[buff_set_pos] = c_end_code[data_pos]
				buff_set_pos++
				set_data_count++
			}
		}
		if set_data_count == c_need_data_len {
			code_size = c_need_data_len
		} else {
			code_size = 0
		}
	} else {
		code_size = 0
	}

	return buff, code_size
}

func getKeyList() map[string][]byte {
	return map[string][]byte{
		"restart":      {0x01, 0x58, 0x00, 0xAE, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x41, 0x00, 0x18, 0x00, 0x14, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x16, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x3E, 0x00, 0x18, 0x1E, 0x0D},
		"fast_back":    {0x01, 0x54, 0x00, 0xAF, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x41, 0x00, 0x17, 0x00, 0x16, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x17, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x3E, 0x00, 0x17, 0x1E, 0x0D},
		"tmp_stop":     {0x01, 0x54, 0x00, 0xB0, 0x00, 0x18, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x14, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x41, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x14, 0x00, 0x17, 0x00, 0x40, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x1B, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x3E, 0x00, 0x18, 0x1E, 0x0D},
		"fast_forward": {0x01, 0x55, 0x00, 0xAE, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x40, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x16, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x16, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x3D, 0x00, 0x19, 0x1E, 0x0D},
		"key_original": {0x01, 0x55, 0x00, 0xAE, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x40, 0x00, 0x19, 0x00, 0x14, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x16, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x3D, 0x00, 0x19, 0x1E, 0x0D},
		"tempo_up":     {0x01, 0x54, 0x00, 0xAF, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x41, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x3E, 0x00, 0x18, 0x1E, 0x0D},
		"tempo_down":   {0x01, 0x55, 0x00, 0xAF, 0x00, 0x18, 0x00, 0x3E, 0x00, 0x1C, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x19, 0x00, 0x40, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x3D, 0x00, 0x19, 0x1E, 0x0D},
		"key_up":       {0x01, 0x54, 0x00, 0xAF, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x14, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x40, 0x00, 0x19, 0x00, 0x14, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x17, 0x00, 0x19, 0x00, 0x14, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x3E, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x40, 0x00, 0x19, 0x00, 0x16, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3E, 0x00, 0x18, 0x1E, 0x0D},
		"key_down":     {0x01, 0x55, 0x00, 0xAF, 0x00, 0x18, 0x00, 0x3E, 0x00, 0x19, 0x00, 0x14, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x41, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x1B, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x19, 0x00, 0x14, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x3D, 0x00, 0x19, 0x1E, 0x0D},
		"stop":         {0x01, 0x58, 0x00, 0xAF, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x17, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x41, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x19, 0x00, 0x15, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x17, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x3F, 0x00, 0x17, 0x00, 0x16, 0x00, 0x18, 0x00, 0x15, 0x00, 0x18, 0x00, 0x16, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x40, 0x00, 0x18, 0x00, 0x3E, 0x00, 0x18, 0x1E, 0x0D},
	}
}
