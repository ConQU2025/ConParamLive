from ConParamLive import ConParamLive
from time import sleep

cpl = ConParamLive(
    namespace="test",
    backend=("localhost", 9165),
    timeout=1,
    param1="value1",
    param2=2
)

while True:
    print(cpl.param1, cpl.param2)
    cpl.param1 += " updated"
    cpl.param2 *= 2
    sleep(1)