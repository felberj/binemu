package mach

import (
	co "github.com/felberj/binemu/kernel/common"
	"reflect"
)

func (k *MachKernel) TaskSelfTrap() uint64 {
	return 1
}

func (k *MachKernel) ThreadSelfTrap() uint64 {
	return 3
}

func (k *MachKernel) MachReplyPort() uint64 {
	return 4
}

func (k *MachKernel) HostSelfTrap() uint64 {
	return 2
}

type mach_msg_header_t struct {
	Bits        uint32 /*msgh_bits*/
	Size        uint32 /*msgh_size*/
	RemotePort  uint32 /*msgh_remote_port*/
	LocalPort   uint32 /*msgh_local_port*/
	VoucherPort uint32 /*msgh_voucher_port*/
	Id          int32  /*msgh_id*/
}

//Network Data Representation record as sent with mach messages generated by mig (mach interface generator)
type ndr_record_t struct {
	Mig_vers     uint8
	If_vers      uint8
	Reserved1    uint8
	Mig_encoding uint8
	Int_rep      uint8
	Char_rep     uint8
	Float_rep    uint8
	Reserved2    uint8
}

type mach_msg_body_t struct {
	Msgh_descriptor_count uint32
}

type mach_msg_port_descriptor_t struct {
	Name        uint32 //TODO: arch dependent and equals size of pointer in newer versions?
	Pad1        uint32
	Pad2        uint16
	Disposition uint8
	Type        uint8
}

type mach_msg_trailer_t struct {
	Msgh_trailer_size uint32
	Msgh_trailer_type uint32
}

type task_get_special_port_request struct {
	NDR   ndr_record_t
	Which int32
}

type task_get_special_port_reply struct {
	Msgh_body    mach_msg_body_t
	Special_port mach_msg_port_descriptor_t
}

type host_info_request struct {
	NDR              ndr_record_t
	Flavor           int32
	Host_info_outCnt uint32
}

type host_info_reply struct {
	NDR              ndr_record_t
	RetCode          int32
	Host_info_outCnt uint32 `struc:"uint32,sizeof=Host_info_out"`
	Host_info_out    []int32
}

type host_get_clock_service_request struct {
	NDR      ndr_record_t
	Clock_id int32
}

type host_get_clock_service_reply struct {
	Msgh_body  mach_msg_body_t
	Clock_serv mach_msg_port_descriptor_t
}

type semaphore_create_request struct {
	NDR    ndr_record_t
	Policy int32
	Value  int32
}

type semaphore_create_reply struct {
	Msgh_body mach_msg_body_t
	Semaphore mach_msg_port_descriptor_t
}

func (k *MachKernel) MachMsgTrap(msg co.Buf, option int32, send_size uint32, rcv_size uint32, rcv_name uint32, timeout uint32) uint64 {
	//rcv_name is also used as notify port when awaiting notification

	//option bits:
	//#define	MACH_SEND_MSG		0x00000001
	//#define	MACH_RCV_MSG		0x00000002

	var header mach_msg_header_t
	err := msg.Unpack(&header)

	msgBody := co.NewBuf(k, msg.Addr+uint64(reflect.TypeOf(header).Size()))

	//MACH_MSGH_BITS(remote, local) = remote | (local << 8)

	switch header.Id {
	case 3409: //task_get_special_port
		var args task_get_special_port_request
		msgBody.Unpack(&args)
		k.U.Printf("args", args)

		header.Id += 100 //reply Id always equals reqId+100

		header.RemotePort = header.LocalPort
		header.LocalPort = 0
		header.Bits &= 0xFF
		//non-simple response structure
		header.Bits |= 0x80000000 //MACH_MSGH_BITS_COMPLEX

		//build reply body
		var reply task_get_special_port_reply
		reply.Msgh_body.Msgh_descriptor_count = 1
		reply.Special_port.Disposition = 17 //meaning?
		reply.Special_port.Type = 0         //typeId MACH_MSG_PORT_DESCRIPTOR = 0
		reply.Special_port.Name = 11        //I just chose 11 randomly here - TODO: properly manage ports

		//adjust size
		header.Size = uint32(reflect.TypeOf(header).Size()) + uint32(reflect.TypeOf(reply).Size())

		//build trailer
		msgTrailer := co.NewBuf(k, msg.Addr+uint64(header.Size))
		var trailer mach_msg_trailer_t
		//TODO: fill trailer - see mach_msg_trailer_t

		//write reply
		msg.Pack(&header)
		msgBody.Pack(&reply)
		msgTrailer.Pack(&trailer)

	case 200: //host_info
		var args host_info_request
		msgBody.Unpack(&args)
		k.U.Printf("args", args)

		header.Id += 100 //reply Id always equals reqId+100

		header.RemotePort = header.LocalPort
		header.LocalPort = 0
		header.Bits &= 0xFF

		//build reply body
		var reply host_info_reply
		reply.NDR = args.NDR
		reply.RetCode = 0 //success
		reply.Host_info_outCnt = 68
		if args.Host_info_outCnt < reply.Host_info_outCnt {
			reply.Host_info_outCnt = args.Host_info_outCnt
		}

		if args.Flavor == 5 { //HOST_PRIORITY_INFO
			//TODO: check for available space?
			reply.Host_info_outCnt = 8 //HOST_PRIORITY_INFO_COUNT

			reply.Host_info_out = make([]int32, args.Host_info_outCnt)

			reply.Host_info_out[0] = 0   //integer_t	kernel_priority;
			reply.Host_info_out[1] = 0   //integer_t	system_priority;
			reply.Host_info_out[2] = 0   //integer_t	server_priority;
			reply.Host_info_out[3] = 0   //integer_t	user_priority;
			reply.Host_info_out[4] = 0   //integer_t	depress_priority;
			reply.Host_info_out[5] = 10  //integer_t	idle_priority;
			reply.Host_info_out[6] = 10  //integer_t	minimum_priority;
			reply.Host_info_out[7] = -10 //integer_t	maximum_priority;
		} else {
			k.U.Printf("unimplemented host_info flavor", args.Flavor)
			panic("host_info not implemented")
		}

		//adjust size
		header.Size = uint32(reflect.TypeOf(header).Size()) + uint32(reflect.TypeOf(reply).Size()) + 4*reply.Host_info_outCnt - uint32(reflect.TypeOf(reply.Host_info_out).Size())

		//build trailer
		msgTrailer := co.NewBuf(k, msg.Addr+uint64(header.Size))
		var trailer mach_msg_trailer_t

		//write reply
		msg.Pack(&header)
		msgBody.Pack(&reply)
		msgTrailer.Pack(&trailer)

	case 206: //host_get_clock_service
		var args host_get_clock_service_request
		msgBody.Unpack(&args)
		k.U.Printf("args", args)

		header.Id += 100 //reply Id always equals reqId+100

		header.RemotePort = header.LocalPort
		header.LocalPort = 0
		header.Bits &= 0xFF

		//non-simple response structure
		header.Bits |= 0x80000000 //MACH_MSGH_BITS_COMPLEX

		//build reply body
		var reply host_get_clock_service_reply
		reply.Msgh_body.Msgh_descriptor_count = 1
		reply.Clock_serv.Disposition = 17 //19//meaning?
		reply.Clock_serv.Type = 0         //typeId MACH_MSG_PORT_DESCRIPTOR = 0
		reply.Clock_serv.Name = 13        //I just chose 13 randomly here - TODO: properly manage ports

		//adjust size
		header.Size = uint32(reflect.TypeOf(header).Size()) + uint32(reflect.TypeOf(reply).Size())

		//build trailer
		msgTrailer := co.NewBuf(k, msg.Addr+uint64(header.Size))
		var trailer mach_msg_trailer_t
		//TODO: fill trailer - see mach_msg_trailer_t

		//write reply
		msg.Pack(&header)
		msgBody.Pack(&reply)
		msgTrailer.Pack(&trailer)

	case 3418: //semaphore_create
		var args semaphore_create_request
		msgBody.Unpack(&args)
		k.U.Printf("args", args)

		header.Id += 100 //reply Id always equals reqId+100

		header.RemotePort = header.LocalPort
		header.LocalPort = 0
		header.Bits &= 0xFF
		//non-simple response structure
		header.Bits |= 0x80000000 //MACH_MSGH_BITS_COMPLEX

		//build reply body
		var reply semaphore_create_reply
		reply.Msgh_body.Msgh_descriptor_count = 1
		reply.Semaphore.Disposition = 17 //19//meaning?
		reply.Semaphore.Type = 0         //typeId MACH_MSG_PORT_DESCRIPTOR = 0
		reply.Semaphore.Name = 14        //I just chose 14 randomly here - TODO: properly manage ports

		//adjust size
		header.Size = uint32(reflect.TypeOf(header).Size()) + uint32(reflect.TypeOf(reply).Size())

		//build trailer
		msgTrailer := co.NewBuf(k, msg.Addr+uint64(header.Size))
		var trailer mach_msg_trailer_t
		//TODO: fill trailer - see mach_msg_trailer_t

		//write reply
		msg.Pack(&header)
		msgBody.Pack(&reply)
		msgTrailer.Pack(&trailer)

	default:
		k.U.Printf("mach msg Id\n", header.Id)
		k.U.Printf("mach msg\n", header, err, option, send_size, rcv_size, rcv_name, timeout)
		panic("mach msg id not implemented")
	}

	return 0
}

func (k *MachKernel) KernelrpcMachPortDeallocateTrap() uint64 {
	//TODO: implement
	return 0
}