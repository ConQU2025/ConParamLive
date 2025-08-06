from socket import socket, AF_INET, SOCK_DGRAM
from json import loads, dumps
from threading import Thread
from time import sleep


class ConParamLive:
    def __recv_daemon(self):
        while True:
            try:
                print("Waiting for data...")
                data, _ = self.__socket.recvfrom(1024)
                self.__parameters.update(loads(data.decode("utf-8")))
            except TimeoutError:
                continue

    def __init__(
        self, namespace="default", backend=("localhost", 9165), timeout=1, **defaults
    ):
        self.__parameters = defaults
        self.__backend = backend

        # 创建 UDP 套接字
        sock = socket(AF_INET, SOCK_DGRAM)
        sock.settimeout(timeout)
        self.__socket = sock
        
        # 启动时向后端发送命名空间数据
        data = dumps({"__namespace__": namespace}).encode("utf-8")
        sock.sendto(data, backend)

        Thread(target=self.__recv_daemon, daemon=True).start()

    def __getattr__(self, name):
        print(f"Getting parameter {name}")
        if name == "_ConParamLive__parameters":
            return object.__getattribute__(self, "__parameters")

        while name not in self.__parameters:
            sleep(0.1)
        return self.__parameters[name]

    def __setattr__(self, name, value):
        if name in ("__backend", "__parameters", "__socket"):
            return object.__setattr__(self, name, value)

        print(f"Setting parameter {name} to value {value}")
        try:
            self.__parameters[name] = value
            data = dumps({name: value}).encode("utf-8")
            print(f"Sending data to backend {self.__backend}: {data}")
            self.__socket.sendto(data, self.__backend)
        except TimeoutError:
            print(
                f"Failed to send parameter {name} with value {value} to backend {self.__backend}"
            )
