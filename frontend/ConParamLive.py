from socket import socket, AF_INET, SOCK_DGRAM
from json import loads, dumps
from threading import Thread, Lock
from time import sleep


class ConParamLive:
    def __recv_daemon(self):
        while True:
            try:
                # print("Waiting for data...")
                data, _ = self.__socket.recvfrom(1024)
                received_data = loads(data.decode("utf-8"))
                with self.__lock:
                    self.__parameters.update(received_data)
            except TimeoutError:
                continue

    def __init__(
        self, namespace="default", backend=("localhost", 9165), timeout=1, **defaults
    ):
        self.__parameters = defaults # 将传入的参数默认值作为初始参数
        self.__backend = backend
        self.__lock = Lock()  # 添加线程锁

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

        while True:
            with self.__lock:
                if name in self.__parameters:
                    return self.__parameters[name]
            sleep(0.1)

    def __setattr__(self, name, value):
        if name in ("__backend", "__parameters", "__socket", "__lock"):
            return object.__setattr__(self, name, value)

        # print(f"Setting parameter {name} to value {value}")
        try:
            with self.__lock:
                self.__parameters[name] = value
            data = dumps({name: value}).encode("utf-8")
            # print(f"Sending data to backend {self.__backend}: {data}")
            self.__socket.sendto(data, self.__backend)
        except TimeoutError:
            pass
