package main

const SAMPLE_CFG = `---
mode:               # 工作模式，必须是"client"或"server"之一
debug: false        # 调试模式（输出更多log信息）
ulimit: 1024        # 最大句柄数量（一般无需调整）
server:             # 服务端配置
  ctrl_port: 3535   # 管理端口（HTTP API）
  serv_port: 35350  # 服务端口（从该端口开始自动分配，第一个用于后端接入，
                    # 后续为用户端接入）
  handshake: 10     # 握手时间窗口（秒）
  idle_close: 0     # 空闲工作连接时效（秒，若为0则使用auth_time）
  auth_time: 86400  # 连接授权最长时限（秒）
  otp_issuer:       # OTP签发机构（仅显示用途，默认为'Door Keeper'）
  users:            # 基于OTP的用户访问控制
    #name: otp-key
  auths:            # 通信密钥组（用于客户端认证）
    #name: shared-key
client:             # 客户端配置
  svr_host:         # 服务端（DKS）的地址（IP或域名）
  svr_port: 35350   # 服务端（DKS）的服务端口
  name:             # 客户端名称
  auth:             # 共享密钥
  lan_nets: []      # 本地网络定义（用于端口扫描，CIDR格式的数组）
  mac_scan: 1000    # 端口扫描时用于扫描MAC地址的超时时间（毫秒，范围100～5000）
logging:
  path: ../log      # LOG文件目录（相对目录基于本配置文件）
  split: 1048576    # 最大LOG字节数（超过则切分）
  keep: 10          # 保留LOG文件数（超过删除最老的）`
