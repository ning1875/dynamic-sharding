# dynamic-sharding 


![image](https://github.com/ning1875/dynamic-sharding/blob/master/images/log.jpg)


`dynamic-sharding`  基于cosul的service健康坚检测实现一致性哈希环动态分片:
一个典型应用场景就是mock pushgateway的 HA(pgw100%的HA实现起来较为困难)

主要解决pgw 单点问题case,实现原理如下: 
- 
- dynamic-sharding服务启动会根据配置文件注册pgw服务到consul中
- 由consul定时对pgw server做http check
- push请求会根据请求path做一致性哈希分离 
- 当多个pgw中实例oom或异常重启,consul会将bad实例标记为down
- dynamic-sharding轮询检查实例数量变化,rehash将job分流
- dynamic-sharding本身无状态,可启动多个实例作为流量接入层和pgw server之间
- 扩容时同时也需要重启所有存量pgw服务



## 使用指南
   

```
# build
$ git clone https://github.com/ning1875/dynamic-sharding.git
$ cd  dynamic-sharding/pkg/ && go build -o dynamic-sharding main.go 

# 修改配置文件
补充dynamic-sharding.yml中的信息:


# 启动dynamic-sharding服务
./dynamic-sharding --config.file=dynamic-sharding.yml

 
# 和promtheus集成 
Add the following text to your promtheus.yaml's scrape_configs section

scrape_configs:
  - job_name: pushgateway
    consul_sd_configs:
      - server: $cousul_api
        services:
          - pushgateway
    relabel_configs:
    - source_labels:  ["__meta_consul_dc"]
      target_label: "dc"



# 调用方调用 dynamic-sharding接口即可 
eg: http://localhost:9292/

```

## 运维指南

### pgw节点故障 (无需关心) 
```apple js
eg: 启动了4个pgw实例,其中一个宕机了,则流量从4->3,以此类推
```

### pgw节点恢复 
```apple js
eg: 启动了4个pgw实例,其中一个宕机了,过一会儿恢复了,那么它会被consul unregister掉
避免出现和扩容一样的case: 再次rehash的job 会持续在原有pgw被prome scrap，而且value不会更新
```


### 扩容
```c
修改yml配置文件将pgw servers 调整到扩容后的数量,重启服务dynamic-sharding 
!! 注意 同时也要重启所有存量pgw服务,不然rehash的job 会持续在原有pgw被prome scrap，而且value不会更新

```

### 缩容

```apple js
# 方法一
## 调用cousul api  
curl -vvv --request PUT 'http://$cousul_api/v1/agent/service/deregister/$pgw_addr_$pgw_port'
eg: curl -vvv --request PUT 'http://localhost:8500/v1/agent/service/deregister/1.1.1.1_9091'

## 修改yml配置文件将pgw servers 调整到缩容后的数量，避免服务重启时再次注册缩容节点

# 方法二
## 停止缩容节点服务,consul会将服务踢出,然后再注销

```

# 交流
## 对本项目感兴趣等可以+我qq:907974064
