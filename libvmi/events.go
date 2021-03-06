package libvmi

import (
  "sync"
  "fmt"
)

/*
#cgo CFLAGS: -Wno-implicit-function-declaration
#cgo LDFLAGS: -lvmi
#include <sys/mman.h>
#include <errno.h>
#include <inttypes.h>
#include <stdlib.h>
#include <libvmi/libvmi.h>
#include <libvmi/events.h>

//the function handler called by all events that will call the go proxy to do a go function callback lookup
 event_response_t generic_event_handler(vmi_instance_t vmi, vmi_event_t *event)
{
  printf("Callback!\n");
  //go_libvmi_event_callback_proxy(vmi,event);
}

//wrap the memset event creation because go has problems with fields in structs and go 1.6 restricts
//go pointers being passed to c

uint64_t
memset_single_step_event(vmi_instance_t vmi,unsigned int version, unsigned int type, unsigned int enable)
{
  vmi_event_t single_event;

  memset(&single_event, 0, sizeof(vmi_event_t));
  single_event.version = version;
  single_event.type = type;
  single_event.callback = generic_event_handler;
  single_event.ss_event.enable = enable;
  SET_VCPU_SINGLESTEP(single_event.ss_event,0);

  //register it
  vmi_register_event(vmi,&single_event);

  uint64_t id = (uintptr_t)&single_event;

  return id;
}
*/
import "C"

const (
  VMI_EVENTS_VERSION = C.VMI_EVENTS_VERSION
  VMI_EVENT_INVALID = C.VMI_EVENT_INVALID
  VMI_EVENT_MEMORY = C.VMI_EVENT_MEMORY
  VMI_EVENT_REGISTER = C.VMI_EVENT_REGISTER
  VMI_EVENT_SINGLESTEP = C.VMI_EVENT_SINGLESTEP
  VMI_EVENT_INTERRUPT = C.VMI_EVENT_INTERRUPT
  VMI_EVENT_GUEST_REQUEST = C.VMI_EVENT_GUEST_REQUEST
  VMI_EVENT_CPUID = C.VMI_EVENT_CPUID
  VMI_EVENT_DEBUG_EXCEPTION = C.VMI_EVENT_DEBUG_EXCEPTION
)

type Libvmi_Event struct{
  event *C.vmi_event_t
  Callback func(Libvmi,Libvmi_Event)
  Version uint
  Type uint
  EnableSingleStepEvent bool
}

func (e *Libvmi_Event) setEvent(event *C.vmi_event_t){
  e.event = event
}



/*
* An explanation of this implementation can be found at the link below. The
* simple version is that we register some key(numeric) that corresponds to a go function pointer
* and pass that key through the C code execution and back to a go proxy function that
* then does a look up on the actual function to call
* https://github.com/golang/go/wiki/cgo#function-variables
*/
var mu sync.Mutex
var fns = make(map[C.uint64_t]Libvmi_Event)

/*
* Register the go callback function in the Libvmi_Event wrapper by using
* the address of the event since libvmi uses the same event struct as a parameter
* to the generic callback function
*/
func register(id C.uint64_t, callback_wrapper Libvmi_Event) {
    mu.Lock()
    defer mu.Unlock()
    fns[id] = callback_wrapper
}

func lookup(id C.uint64_t) Libvmi_Event {
    mu.Lock()
    defer mu.Unlock()
    return fns[id]
}

func unregister(id C.uint64_t) {
    mu.Lock()
    defer mu.Unlock()
    delete(fns, id)
}


func Vmi_events_listen(vmi Libvmi,timeout uint32){
  C.vmi_events_listen(vmi.vmi,(C.uint32_t)(timeout))
}

func Vmi_register_event(vmi Libvmi, event Libvmi_Event){
  switch event.Type {
  case VMI_EVENT_SINGLESTEP:
    enable := 0
    if event.EnableSingleStepEvent == true{
      enable = 1
    }

    id := C.memset_single_step_event(vmi.vmi,C.uint(event.Version),C.uint(event.Type),C.uint(enable))
    //register the struct address so we can lookup the callback in the map later
    register(id,event)

  default:
    fmt.Println("Unknown event type")
    return;
  }
}
