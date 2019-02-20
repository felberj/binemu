package x86_64

import (
	uc "github.com/felberj/binemu/cpu/unicorn"
	"github.com/lunixbochs/ghostrace/ghost/sys/num"

	"github.com/felberj/binemu/kernel/common"
	"github.com/felberj/binemu/models"
)

type NameMapEntry struct {
	id     int
	subMap map[string]NameMapEntry
}

var SysctlNameMapKern = map[string]NameMapEntry{
	"ostype":          {1, nil},
	"osrelease":       {2, nil},
	"osrevision":      {3, nil},
	"version":         {4, nil},
	"maxvnodes":       {5, nil},
	"maxproc":         {6, nil},
	"maxfiles":        {7, nil},
	"argmax":          {8, nil},
	"securelevel":     {9, nil},
	"hostname":        {10, nil},
	"hostid":          {11, nil},
	"clockrate":       {12, nil},
	"vnode":           {13, nil},
	"proc":            {14, nil},
	"file":            {15, nil},
	"profiling":       {16, nil},
	"posix1version":   {17, nil},
	"ngroups":         {18, nil},
	"job_control":     {19, nil},
	"saved_ids":       {20, nil},
	"boottime":        {21, nil},
	"nisdomainname":   {22, nil},
	"maxpartitions":   {23, nil},
	"kdebug":          {24, nil},
	"update":          {25, nil},
	"osreldate":       {26, nil},
	"ntp_pll":         {27, nil},
	"bootfile":        {28, nil},
	"maxfilesperproc": {29, nil},
	"maxprocperuid":   {30, nil},
	"dumpdev":         {31, nil}, /* we lie; don't print as int */
	"ipc":             {32, nil},

	"usrstack":   {35, nil},
	"logsigexit": {36, nil},
	"symfile":    {37, nil},
	"procargs":   {38, nil},

	"netboot":   {40, nil},
	"panicinfo": {41, nil},
	"sysv":      {42, nil},

	"exec":           {45, nil},
	"aiomax":         {46, nil},
	"aioprocmax":     {47, nil},
	"aiothreads":     {48, nil},
	"procargs2":      {49, nil},
	"corefile":       {50, nil},
	"coredump":       {51, nil},
	"sugid_coredump": {52, nil},
	"delayterm":      {53, nil},
	"shreg_private":  {54, nil},

	"low_pri_window":             {56, nil},
	"low_pri_delay":              {57, nil},
	"posix":                      {58, nil},
	"usrstack64":                 {59, nil},
	"nx":                         {60, nil},
	"tfp":                        {61, nil},
	"procname":                   {62, nil},
	"threadsigaltstack":          {63, nil},
	"speculative_reads_disabled": {64, nil},
	"osversion":                  {65, nil},
	"safeboot":                   {66, nil},
	"lctx":                       {67, nil},
	"rage_vnode":                 {68, nil},
	"tty":                        {69, nil},
	"check_openevt":              {70, nil},
	"thread_name":                {71, nil},
}

var SysctlNameMapVfs = map[string]NameMapEntry{
	"vfsconf": {0, nil},
}

var SysctlNameMapVm = map[string]NameMapEntry{
	"vmmeter": {1, nil},
	"loadavg": {2, nil},

	"swapusage": {5, nil},
}

var SysctlNameMapHw = map[string]NameMapEntry{
	"machine":       {1, nil},
	"model":         {2, nil},
	"ncpu":          {3, nil},
	"byteorder":     {4, nil},
	"physmem":       {5, nil},
	"usermem":       {6, nil},
	"pagesize":      {7, nil},
	"disknames":     {8, nil},
	"diskstats":     {9, nil},
	"epoch":         {10, nil},
	"floatingpoint": {11, nil},
	"machinearch":   {12, nil},
	"vectorunit":    {13, nil},
	"busfrequency":  {14, nil},
	"cpufrequency":  {15, nil},
	"cachelinesize": {16, nil},
	"l1icachesize":  {17, nil},
	"l1dcachesize":  {18, nil},
	"l2settings":    {19, nil},
	"l2cachesize":   {20, nil},
	"l3settings":    {21, nil},
	"l3cachesize":   {22, nil},
	"tbfrequency":   {23, nil},
	"memsize":       {24, nil},
	"availcpu":      {25, nil},
}

var SysctlNameMapUser = map[string]NameMapEntry{
	"cs_path":          {1, nil},
	"bc_base_max":      {2, nil},
	"bc_dim_max":       {3, nil},
	"bc_scale_max":     {4, nil},
	"bc_string_max":    {5, nil},
	"coll_weights_max": {6, nil},
	"expr_nest_max":    {7, nil},
	"line_max":         {8, nil},
	"re_dup_max":       {9, nil},
	"posix2_version":   {10, nil},
	"posix2_c_bind":    {11, nil},
	"posix2_c_dev":     {12, nil},
	"posix2_char_term": {13, nil},
	"posix2_fort_dev":  {14, nil},
	"posix2_fort_run":  {15, nil},
	"posix2_localedef": {16, nil},
	"posix2_sw_dev":    {17, nil},
	"posix2_upe":       {18, nil},
	"stream_max":       {19, nil},
	"tzname_max":       {20, nil},
}

var SysctlNameMapCTL = map[string]NameMapEntry{
	"kern":    {1, SysctlNameMapKern},
	"vm":      {2, SysctlNameMapVm},
	"vfs":     {3, SysctlNameMapVfs},
	"net":     {4, nil},
	"debug":   {5, nil},
	"hw":      {6, SysctlNameMapHw},
	"machdep": {7, nil},
	"user":    {8, SysctlNameMapUser},
}

func DarwinSyscall(u models.Usercorn) {
	rax, _ := u.RegRead(uc.X86_REG_RAX)
	name, _ := num.Darwin_x86_mach[int(rax)]
	ret, _ := u.Syscall(int(rax), name, common.RegArgs(u, AbiRegs))
	u.RegWrite(uc.X86_REG_RAX, ret)
}

func DarwinInterrupt(u models.Usercorn, intno uint32) {
	if intno == 0x80 {
		DarwinSyscall(u)
	}
}
