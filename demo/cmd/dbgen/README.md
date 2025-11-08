# dbgen

作用: 在执行`run.py`结束，批量编译项目完成，在`statRoot`目录下生成`csv`和`graphs`目录之后，使用如下命令将逃逸根因信息导入数据库。

```bash
go run . <yaml config file, bench.yaml for example>
```

## schema

数据维度：

- package path
- batch number, 一个batch内可能分析多个函数
- 结点id
- 结点逃逸根因
  - 一个结点的逃逸根因可能有多个
- 结点逃逸根因数量
  - 根据结点逃逸根因统计的结果，方便加速查询
- 结点位置
  - 一个结点只有一个位置。

package path+batch number+结点id可以唯一确定一个结点

数据库schema设计：

两个table:
1. node_info
   1. package_path
   2. batch_number
   3. node_id
   4. node_location
   5. root_cause_count
   6. uid (unique id，作为主键)
2. node_root_causes (uid和root_cause_id一起作为主键)
   1. uid
   2. root_cause_id (Enum类型，支持使用`'ERHeap'`字符串匹配查询)

## 端口转发

可以通过转发8123端口在本地使用web sql查询

## 查询

注意在数据表名前需要添加数据库名

下图为改造后的 [paramleak](./sql/paramleak.sql) 查询：

![](figures/modified_paramleak_sql.png)

查询结果可以直接复制HTML到excel中。


