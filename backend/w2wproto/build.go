//go:build exclude

package w2wproto

/*


#cgo LDFLAGS: -L/Users/kzz/kUser/goUser/lib -lprotobuf
#cgo CPPFLAGS: -I${SRCDIR}/../ -std=c17 -I/Users/kzz/kUser/goUser/include
#cgo CXXFLAGS: -I${SRCDIR}/../ -std=c++14 -I/Users/kzz/kUser/goUser/include

#include "w2wservice.pb.h"
*/
import "C"

func buildStub() {

}
