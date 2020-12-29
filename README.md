
## xdp-redirect demo

An epbf program to demonstrate how the XDP redirect network packets

- [xdp-redirect demo](#xdp-redirect-demo)
- [Prerequisites](#prerequisites)
  - [Prepare development environment](#prepare-development-environment)
  - [Background](#background)
- [Get Started](#get-started)
  - [How to build the program](#how-to-build-the-program)
  - [How to run the demo program](#how-to-run-the-demo-program)
  - [(Un)Load the ebpf program to the backend](#unload-the-ebpf-program-to-the-backend)
  - [Start the web server](#start-the-web-server)
  - [Start a container to accept redirected packets](#start-a-container-to-accept-redirected-packets)
  - [Configuring the redirecting rules](#configuring-the-redirecting-rules)
  - [Redirecting packets](#redirecting-packets)
    - [Observe packets redirecting statistics](#observe-packets-redirecting-statistics)
    - [Redirecting packets](#redirecting-packets-1)



## Prerequisites

### Prepare development environment

In order to prepare the eBPF development environment, you can refer to [BPF and XDP Reference Guide](https://docs.cilium.io/en/stable/bpf/#development-environment) of the Cilium document 


### Background

The program demonstrates the XDP redirects UDP packages to another network interface. The following diagram illustrates the background in which the demo was run.

![background](png/background.png)

There are five components that are responsible for different roles in the demo:

- **Web server**: A web server implemented in Golang language, which is dedicated to providing API for users to maintain redirecting rules for the XDP program.
- **A virtual machine**: A KVM hypervisor-based virtual machine, the guest os is Linux, which runs a netcat program, sending UDP requests from the virtual machine to the docker container of the host.
- **XDP**: an eBPF program, which attaches to the virtual machine's bridge interface (virbr0), triggered by incoming packets. When arrived packets are the UDP protocol and the destination port is `7999`, the XDP program will redirects packets to a specific interface according to redirecting rules in the eBPF map.
- **Docker container**: a container is dedicated to receiving packets redirected by the XDP program. the redirected packets are sent to UDP server in the container via the container's veth pair interface.
- **Redirecting rules**: Redirecting rules are an array that contains Mac, IP address, and interface index. When The XDP program is triggered, it deicides to chose information of the rule to redirect the packet.

## Get Started

### How to build the program

After cloning this repository, you need to build the web server programming and eBPF program. 

- **Build web server**: In the root directory of the repository, run make in the shell environment. The target binary `xdplbmgmt` will be built in the `bin/` directory.

> The web server is implemented in the Golang language, you need to prepare the Golang development environment in advance.


- **Build eBPF program**: In the root directory of the repository, run `git submodule update --init` command to update `libbpf` sub module. After the `libbpf` module is downloaded, enter the `ebpf` directory, run `make` to build the ebpf target. the ebpf target `xdp_redirect.o` is generated in the current directory (`ebpf/`)

### How to run the demo program

### (Un)Load the ebpf program to the backend

You can load ebpf program via the iproute2 command. I provide simple scripts to load/unload the ebpf program.  Assuming your bridge network device name of the hypervisor is `virbr0`, 
running following command in the root directory of the repository to load the ebpf program to device `virbr0`:

```shell
$ sudo sbin/load.sh virbr0
```

When the demonstration has finished, using the following command to detach the ebpf program

```shell
$ sudo sbin/unload.sh virbr0
```

### Start the web server

After building web server, the executable is placed in `bin/` directory. You can start it via the following command:

```shell
$ nohup sudo bin/xdplbmgmt &
```
The server listens on `9091` port by default. You can change the listening address via `--address` parameter. For example:

```shell
$ nohup sudo bin/xdplbmgmt --address :9093
```
The web server will listen 9093 port on all interfaces

After demonstration has finished, stop web server via:

```shell
$ sudo killall xdplbmgmt
```

### Start a container to accept redirected packets

A helper container is dedicated to accepting redirected packets. Starting a container listen 7999 port of UDP via the following command

```shell
$ docker run --name udpserver --rm -it nicolaka/netshoot  nc -kul 172.17.0.2 7999 
```

Checking the IP and MAC address of the container's `eth0` interface  

```shell
$ docker exec udpserver ip a show dev eth0 
24: eth0@if25: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default
    link/ether 02:42:ac:11:00:02 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 172.17.0.2/16 brd 172.17.255.255 scope global eth0
       valid_lft forever preferred_lft forever
```
You may notice the IP address is: `172.17.0.2` and MAC address is: `02:42:ac:11:00:02`. All address values will be configured in the redirect map, you'd better save them somewhere.

After the demonstration finished, stop the container via:
```shell
$ docker stop udpserver
```

Except for interface `eth0` address in the container, you need to check the interface index of veth pair against `eth0`

Running `ip a` on the host, the last item of interfaces should be another interface of `veth` pair. the output of the `IP a` command should like below:

```
....
21: vnet0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel master virbr0 state UNKNOWN group default qlen 1000
    link/ether 52:5d:6a:08:42:32 brd ff:ff:ff:ff:ff:ff
    inet6 fe80::505d:6aff:fe08:4232/64 scope link
       valid_lft forever preferred_lft forever
25: vethea881ef@if24: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue master docker0 state UP group default
    link/ether ee:1d:d4:fd:50:55 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet6 fe80::ec1d:d4ff:fefd:5055/64 scope link
       valid_lft forever preferred_lft forever
```
`25` is the interface index of veth pair

The value of the following table will be configured in the eBPF map via the web server

| Item | Value|
|-|-|
| IP address of the container | 172.17.0.2|
| MAC address of the container | 02:42:ac:11:00:02|
| Interface index of veth pair | 25 | 

### Configuring the redirecting rules

The `POST /rules` interface of the web server is used to configured the redirecting rule in the eBPF map.

As you have got the IP and MAC address of the container and interface index of the veth pair, you can send a request to `POST /rules` of the web server.

The following command helps you configuring rules

```shell
$ curl -v -XPOST \
  -d '[{"server": "172.17.0.2", "mac": "02:42:ac:11:00:02", "ifindex": 25}]' \
  -H "Content-Type: application/json" \
  http://127.0.0.1:9091/rules\?sourceAddr\=[virtual machine address]
```

### Redirecting packets

#### Observe packets redirecting statistics

The web server provides `GET /rules` interface to observe packet redirecting statistics, the following pieces of script will help you accessing `GET /rules` API periodically

```shell
$ while true
do
sleep 1
clear && curl localhost:9091/rules 2>/dev/null |jq --color-output
done
```

The JSON outputs of the script will be refreshed periodically.
```json
[
  {
    "server": "172.17.0.2",
    "mac": "02:42:ac:11:00:02",
    "ifindex": 25,
    "forward_bytes": 0,
    "forward_packages": 0
  }
]
```
When packets are redirected, values of the `forward_bytes` and `forward_packages` will be changed.

#### Redirecting packets

- Start the virtual machine and login into it via ssh. 
- Using the following command to begin sending UDP packet to an unreachable address

```shell
$ nc -u 11.22.33.44.55 7999
hello ebpf 

```
You may notice that `hello ebpf` will be displayed in the console of the docker container, and the values of `forward_bytes` and `forward_packages` of the observing script will be changed also.

