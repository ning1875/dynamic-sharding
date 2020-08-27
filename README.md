# pgw是什么
[项目介绍](https://github.com/prometheus/pushgateway)
## pgw打点特点

- 没有使用grouping对应的接口uri为 
```
http://pushgateway_addr/metrics/job/<JOB_NAME>
```
- 使用grouping对应的接口uri为 
```
http://pushgateway_addr/metrics/job/<JOB_NAME>/<LABEL_NAME>/<LABEL_VALUE>
```
- put/post方法区别在于 put只替换metrics和job相同的 post替换label全部相同的
# pgw单点问题
## 如果简单把pgw挂在lb后面的问题
- lb后面rr轮询:如果不加控制的让push数据随机打到多个pushgateway实例上,prometheus无差别scrape会导致数据错乱,表现如下
![image](https://github.com/ning1875/dynamic-sharding/blob/master/images/pgw_miss.png)
![image](https://github.com/ning1875/dynamic-sharding/blob/master/images/pgw_miss2.png)
- 根本原因是在t1时刻 指标的值为10 t2时刻 值为20
- t1时刻轮询打点到了pgw-a上 t2时刻打点到了pgw-b上
- 而promethues采集的时候两边全都采集导致本应该一直上升的值呈锯齿状
## 如果对uri做静态一致性哈希+prome静态配置pgw
- 假设有3个pgw,前面lb根据request_uri做一致性哈希
- promethues scrape时静态配置3个pgw实例
```
  - job_name: pushgateway
    honor_labels: true
    honor_timestamps: true
    scrape_interval: 5s
    scrape_timeout: 4s
    metrics_path: /metrics
    scheme: http
    static_configs:
    - targets:
      - pgw-a:9091
      - pgw-b:9091

```
- 结果是可以做到哈希分流,但无法解决某个pgw实例挂掉,哈希到这个实例上面的请求失败问题
## 解决方案是: 动态一致性哈希分流+consul service_check
![image](https://github.com/ning1875/dynamic-sharding/blob/master/images/log.jpg)
- dynamic-sharding服务启动会根据配置文件注册pgw服务到consul中
- 由consul定时对pgw server做http check
- push请求会根据请求path做一致性哈希分离,eg:
```
# 仅job不同
- http://pushgateway_addr/metrics/job/job_a
- http://pushgateway_addr/metrics/job/job_b
- http://pushgateway_addr/metrics/job/job_c
# label不同
- http://pushgateway_addr/metrics/job/job_a/tag_a/value_a
- http://pushgateway_addr/metrics/job/job_a/tag_a/value_b
```
- 当多个pgw中实例oom或异常重启,consul check service会将bad实例标记为down
~~- dynamic-sharding轮询检查实例数量变化~~
- dynamic-sharding 会`Watch` pgw节点数量变化
- 重新生成哈希环,rehash将job分流
- 同时promethues使用consul服务发现的pgw实例列表,无需手动变更
- 采用redirect而不处理请求,简单高效
- dynamic-sharding本身无状态,可启动多个实例作为流量接入层和pgw server之间
- 扩容时同时也需要重启所有存量pgw服务
- 不足:没有解决promethues单点问题和分片问题
项目地址: [https://github.com/ning1875/dynamic-sharding](https://github.com/ning1875/dynamic-sharding)

## 使用指南
   

```c
build
$ git clone https://github.com/ning1875/dynamic-sharding.git
$ cd  dynamic-sharding/pkg/ && go build -o dynamic-sharding main.go 


修改配置文件
补充dynamic-sharding.yml中的信息:


启动dynamic-sharding服务
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


```