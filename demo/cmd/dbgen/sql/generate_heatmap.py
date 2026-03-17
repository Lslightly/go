#!/usr/bin/env python3
"""
非堆根因统计热力图生成脚本

功能：
- 从CSV文件中读取非堆根因统计的count和percentage数据
- 生成两个热力图：count热力图和percentage热力图
- 自动处理CSV文件中的空行分隔符和标题标记

使用方法：
    python3 generate_heatmap.py --count-file <count_csv> --percentage-file <percentage_csv>

输出文件：
- nonheap_root_causes_count_heatmap.png
- nonheap_root_causes_percentage_heatmap.png
"""

import argparse
import os
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import numpy as np


def parse_csv_simple(csv_file):
    """
    解析简单的CSV文件（不包含标题标记）
    
    CSV文件格式：
    - 第一行是表头
    - 后续行是数据
    
    参数:
        csv_file: CSV文件路径
        
    返回:
        DataFrame: 解析后的数据
    """
    # 直接使用pandas读取CSV文件
    df = pd.read_csv(csv_file)
    return df


def clean_dataframe(df):
    """
    清理DataFrame，移除百分号并转换为数值类型
    
    参数:
        df: 原始DataFrame
        
    返回:
        DataFrame: 清理后的DataFrame
    """
    # 创建DataFrame的副本
    df_clean = df.copy()
    
    # 获取数据列（排除第一列，通常是标签列）
    data_columns = df_clean.columns[1:]
    
    # 清理数据列
    for col in data_columns:
        # 移除百分号并转换为float
        df_clean[col] = df_clean[col].astype(str).str.replace('%', '').str.replace(' ', '')
        # 尝试转换为数值，无法转换的保持为0
        df_clean[col] = pd.to_numeric(df_clean[col], errors='coerce').fillna(0)
    
    return df_clean


def create_heatmap(df, title, output_file, cmap='YlOrRd', figsize=(12, 8)):
    """
    创建热力图
    
    参数:
        df: 数据DataFrame
        title: 图表标题
        output_file: 输出文件路径
        cmap: 颜色映射（默认使用黄橙红色系）
        figsize: 图表大小
    """
    # 设置图表大小
    plt.figure(figsize=figsize)
    
    # 创建热力图
    sns.heatmap(
        df.iloc[:, 1:].values,  # 使用数值数据（排除第一列标签）
        annot=True,             # 显示数值
        fmt='.2f',              # 数值格式
        cmap=cmap,              # 颜色映射
        cbar=True,              # 显示颜色条
        square=False,           # 单元格可以是矩形
        linewidths=0.5,        # 单元格边框线宽
        xticklabels=df.columns[1:],  # X轴标签
        yticklabels=df.iloc[:, 0],   # Y轴标签
        annot_kws={'size': 10}  # 注释字体大小
    )
    
    # 设置标题和标签
    # plt.title(title, fontsize=14, fontweight='bold', pad=20) # ignore title for latex
    plt.xlabel('仓库', fontsize=12)
    plt.ylabel('根因', fontsize=12)
    
    # 旋转X轴标签以避免重叠
    plt.xticks(rotation=45, ha='right')
    plt.yticks(rotation=0)
    
    # 调整布局
    plt.tight_layout()
    
    # 保存图表
    plt.savefig(output_file, dpi=300, bbox_inches='tight')
    plt.close()
    
    print(f"已生成热力图: {output_file}")


def main():
    """主函数"""
    # 解析命令行参数
    parser = argparse.ArgumentParser(
        description='生成非堆根因统计热力图',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog='''
示例:
    python3 generate_heatmap.py --count-file nonheap_root_causes_count.csv --percentage-file nonheap_root_causes_percentage.csv
        '''
    )
    
    parser.add_argument(
        '--count-file',
        required=True,
        help='包含count数据的CSV文件路径'
    )
    
    parser.add_argument(
        '--percentage-file',
        required=True,
        help='包含percentage数据的CSV文件路径'
    )
    
    parser.add_argument(
        '--output-dir',
        default=None,
        help='输出目录（默认与CSV文件相同）'
    )
    
    args = parser.parse_args()
    
    # 确定输出目录
    if args.output_dir:
        output_dir = args.output_dir
        os.makedirs(output_dir, exist_ok=True)
    else:
        # 使用count文件的目录作为输出目录
        output_dir = os.path.dirname(args.count_file) or '.'
    
    # ========================================================================
    # 处理Count数据
    # ========================================================================
    print("处理Count数据...")
    
    # 解析CSV文件
    count_df = parse_csv_simple(args.count_file)
    
    # 清理数据
    count_df_clean = clean_dataframe(count_df)
    
    # 生成Count热力图
    count_output = os.path.join(output_dir, 'nonheap_root_causes_count_heatmap.png')
    create_heatmap(
        count_df_clean,
        '栈分配结点根因分布',
        count_output,
        cmap='YlOrRd'
    )
    
    # ========================================================================
    # 处理Percentage数据
    # ========================================================================
    print("\n处理Percentage数据...")
    
    # 解析CSV文件
    percentage_df = parse_csv_simple(args.percentage_file)
    
    # 清理数据
    percentage_df_clean = clean_dataframe(percentage_df)
    
    # 生成Percentage热力图
    percentage_output = os.path.join(output_dir, 'nonheap_root_causes_percentage_heatmap.png')
    create_heatmap(
        percentage_df_clean,
        '栈分配结点逃逸根因分布（百分比）',
        percentage_output,
        cmap='YlGnBu'
    )
    
    # ========================================================================
    # 完成
    # ========================================================================
    print("\n热力图生成完成！")
    print(f"输出目录: {output_dir}")
    print(f"  - {count_output}")
    print(f"  - {percentage_output}")


if __name__ == '__main__':
    main()