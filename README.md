
## xdp-redirect demo

An epbf program to demonstrate how the XDP redirect network packets

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
sudo sbin/load.sh virbr0
```

When the demonstration has finished, using the following command to detach the ebpf program

```shell
sudo sbin/unload.sh virbr0
```

### Start the web server

After building web server, the executable is placed in `bin/` directory. You can start it via the following command:

```shell
nohup sudo bin/xdplbmgmt &
```
The server listens on `9091` port by default. You can change the listening address via `--address` parameter. For example:

```shell
nohup sudo bin/xdplbmgmt --address :9093
```
The web server will listen 9093 port on all interfaces

After demonstration has finished, stop web server via:

```shell
sudo killall xdplbmgmt
```

### Start a container to accept redirected packets

A helper container is dedicated to accepting redirected packets. Starting a container listen 7999 port of UDP via the following command

```shell
docker run --name udpserver --rm -it nicolaka/netshoot  nc -kul 172.17.0.2 7999 
```

Checking the IP and MAC address of the container's `eth0` interface  

```shell
> docker exec udpserver ip a show dev eth0 
24: eth0@if25: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default
    link/ether 02:42:ac:11:00:02 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 172.17.0.2/16 brd 172.17.255.255 scope global eth0
       valid_lft forever preferred_lft forever
```
You may notice the IP address is: `172.17.0.2` and MAC address is: `02:42:ac:11:00:02`. All address values will be configured in the redirect map, you'd better save them somewhere.

After the demonstration finished, stop the container via:
```shell
docker stop udpserver
```
