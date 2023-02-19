package rtc

const (
	HEVC_NAL_TRAIL_N        = 0
	HEVC_NAL_TRAIL_R        = 1 //
	HEVC_NAL_TSA_N          = 2
	HEVC_NAL_TSA_R          = 3
	HEVC_NAL_STSA_N         = 4
	HEVC_NAL_STSA_R         = 5
	HEVC_NAL_RADL_N         = 6
	HEVC_NAL_RADL_R         = 7
	HEVC_NAL_RASL_N         = 8
	HEVC_NAL_RASL_R         = 9
	HEVC_NAL_VCL_N10        = 10
	HEVC_NAL_VCL_R11        = 11
	HEVC_NAL_VCL_N12        = 12
	HEVC_NAL_VCL_R13        = 13
	HEVC_NAL_VCL_N14        = 14
	HEVC_NAL_VCL_R15        = 15
	HEVC_NAL_BLA_W_LP       = 16
	HEVC_NAL_BLA_W_RADL     = 17
	HEVC_NAL_BLA_N_LP       = 18
	HEVC_NAL_IDR_W_RADL     = 19
	HEVC_NAL_IDR_N_LP       = 20
	HEVC_NAL_CRA_NUT        = 21
	HEVC_NAL_RSV_IRAP_VCL22 = 22
	HEVC_NAL_RSV_IRAP_VCL23 = 23
	HEVC_NAL_RSV_VCL24      = 24
	HEVC_NAL_RSV_VCL25      = 25
	HEVC_NAL_RSV_VCL26      = 26
	HEVC_NAL_RSV_VCL27      = 27
	HEVC_NAL_RSV_VCL28      = 28
	HEVC_NAL_RSV_VCL29      = 29
	HEVC_NAL_RSV_VCL30      = 30
	HEVC_NAL_RSV_VCL31      = 31
	HEVC_NAL_VPS            = 32
	HEVC_NAL_SPS            = 33
	HEVC_NAL_PPS            = 34
	HEVC_NAL_AUD            = 35
	HEVC_NAL_EOS_NUT        = 36
	HEVC_NAL_EOB_NUT        = 37
	HEVC_NAL_FD_NUT         = 38
	HEVC_NAL_SEI_PREFIX     = 39
	HEVC_NAL_SEI_SUFFIX     = 40
	HEVC_NAL_RSV_NVCL41     = 41
	HEVC_NAL_RSV_NVCL42     = 42
	HEVC_NAL_RSV_NVCL43     = 43
	HEVC_NAL_RSV_NVCL44     = 44
	HEVC_NAL_RSV_NVCL45     = 45
	HEVC_NAL_RSV_NVCL46     = 46
	HEVC_NAL_RSV_NVCL47     = 47
	HEVC_NAL_UNSPEC48       = 48
	HEVC_NAL_UNSPEC49       = 49
	HEVC_NAL_UNSPEC50       = 50
	HEVC_NAL_UNSPEC51       = 51
	HEVC_NAL_UNSPEC52       = 52
	HEVC_NAL_UNSPEC53       = 53
	HEVC_NAL_UNSPEC54       = 54
	HEVC_NAL_UNSPEC55       = 55
	HEVC_NAL_UNSPEC56       = 56
	HEVC_NAL_UNSPEC57       = 57
	HEVC_NAL_UNSPEC58       = 58
	HEVC_NAL_UNSPEC59       = 59
	HEVC_NAL_UNSPEC60       = 60
	HEVC_NAL_UNSPEC61       = 61
	HEVC_NAL_UNSPEC62       = 62
	HEVC_NAL_UNSPEC63       = 63
)

type Nalu struct {
	Type    int
	Dts     uint32
	Payload []uint8
}

type Hevc struct {
	ps       []byte
	naluList []Nalu
}

func H265Parser(data []byte) (hevc Hevc) {
	preIndex := 0
	index := 0
	dataLen := len(data)
	var dts uint32 = 0
	//分帧
	for {
		if index+5 >= dataLen {
			nalu := Nalu{}
			nalu.Payload = make([]byte, dataLen-preIndex)
			copy(nalu.Payload, data[preIndex:])
			hevc.naluList = append(hevc.naluList, nalu)
			break
		}

		if data[index] == 0 && data[index+1] == 0 && data[index+2] == 0 &&
			data[index+3] == 1 && ((data[index+4]&0x7E)>>1 == HEVC_NAL_VPS ||
			(data[index+4]&0x7E)>>1 == HEVC_NAL_TRAIL_R) && data[index+5] == 1 {
			if index != 0 {
				nalu := Nalu{
					Dts: dts,
				}
				if (data[index+4]&0x7E)>>1 == HEVC_NAL_VPS {
					nalu.Type = HEVC_NAL_VPS
				} else {
					nalu.Type = HEVC_NAL_TRAIL_R
				}
				nalu.Payload = make([]byte, index-preIndex)
				copy(nalu.Payload, data[preIndex:index])
				hevc.naluList = append(hevc.naluList, nalu)
				preIndex = index
				dts += 40
			}
		}
		index++
	}

	//解析vps&sps&pps
	firstFrame := hevc.naluList[0].Payload
	for index = 0; index < len(firstFrame); index++ {
		if firstFrame[index] == 0 && firstFrame[index+1] == 0 && firstFrame[index+2] == 0 &&
			firstFrame[index+3] == 1 &&
			(data[index+4]&0x7E)>>1 == HEVC_NAL_IDR_W_RADL &&
			data[index+5] == 1 {
			hevc.ps = make([]byte, index)
			copy(hevc.ps, firstFrame[:index])
			break
		}
	}
	return
}
