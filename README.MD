作为集群的基础单位，此服务起到了至关重要的作用
1、中转逻辑
2、其他逻辑
3、DB逻辑

1、中心， - 网关 - 其他逻辑 
2、数据包装， 在网关中不存在解析原本数据包的逻辑代码，这样可以对任何类型的通讯协议都可以使用。也可以使用C或者C++进行编写
3、数据库， 主要是保存一些缓存数据， 查询数据库功能

逻辑关系：
    中心            数据库服务器
        其他服务器 --- ---
    网关

调度服务器，都需要连接到中心服务器后才能进行相互交换数据。