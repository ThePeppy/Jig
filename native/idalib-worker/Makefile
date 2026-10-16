# Headers: git clone --depth 1 --filter=blob:none --sparse -b v9.3.0-release https://github.com/HexRaysSA/ida-sdk.git ../../third_party/ida-sdk && git -C ../../third_party/ida-sdk sparse-checkout set src/include

IDA ?= /Applications/IDA Professional 9.3.app/Contents/MacOS
SDK := ../../third_party/ida-sdk
CXX ?= clang++

# mygo build copies this binary into the app (mygo.json resources).

jig-idalib: main.cpp tools.cpp json.hpp tools.hpp
	$(CXX) -std=c++17 -O2 -D__EA64__ \
		-I$(SDK)/src/include \
		-L"$(IDA)" -lida -lidalib \
		-Wl,-rpath,"$(IDA)" \
		-o $@ main.cpp tools.cpp
