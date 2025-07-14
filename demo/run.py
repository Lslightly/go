import argparse
import subprocess
from pathlib import Path
from typing import Callable, List
from define import EscapeReason
import networkx as nx
import matplotlib.pyplot as plt
from contextlib import redirect_stdout

import pandas as pd

dry_run = False
do_build = False
demo_only = False

def execute(cmd: str, cwd: str | None = None, stdout=None, stderr=None):
    if dry_run:
        print("pwd:", cwd, "cmd:`", cmd, '`')
        return 0, "", ""
    proc = subprocess.run(cmd,
            shell=True, text=True,
            cwd=cwd,
            stdout=stdout, stderr=stderr)
    return proc.returncode, proc.stdout, proc.stderr


def build(repo: Path, EAResultRoot: Path):
    """ Use 'go build -a -gcflags="all=-m=2" .' command
        to generate graphs and csv under EAResultRoot, i.e.
            EAResultRoot/
                graphs/
                    pkg1/import/path
                        0/
                            0xxx.gv 0xxx.svg 1yyy.gv 1yyy.svg
                        1/
                            ...
                    pkg2/import/path
                        ...
                csv/
                    pkg1/import/path
                        all.csv
                        nodeReasonCnt.csv
                        reason2cnt.csv
                    ...
    Args:
        dir (str): source dir
        EAResultRoot (str): root of escape analysis result
    """
    print(f"Building {repo}...")
    # bugs still exist
    # execute(f'go build -a -gcflags="all=-m=2 -earesultroot={EAResultRoot}" .', cwd=str(repo))
    execute(f'go build -a -gcflags="all=-eareasonstat -earesultroot={EAResultRoot}" .', cwd=str(repo))

def collect(EAResultRoot: Path) -> bool:
    """ Collect all all.csv from EAResultRoot/csv.
        The result is written under EAResultRoot/csv.

        For each package import path, create a new DataFrame by adding "pkg" column in DataFrame of pkg/import/path/all.csv using the relative path to EAResultRoot and dump the collected all packages' DataFrame in EAResultRoot/csv/all.csv

        return False if there is no all.csv
    """
    print(f"Collecting {EAResultRoot}...")
    csv_root = EAResultRoot/"csv"
    all_dfs: List[pd.DataFrame] = []
    for csv_path in csv_root.rglob("all.csv"):
        # 计算包的相对路径
        pkg_path = csv_path.parent.relative_to(csv_root)
        # 读取csv
        df = pd.read_csv(csv_path)
        df["pkg"] = str(pkg_path)
        all_dfs.append(df)
    if all_dfs:
        all_df = pd.concat(all_dfs, ignore_index=True)
        all_df.to_csv(EAResultRoot / "all.csv", index=False)
        return True
    else:
        print(f"No all.csv files found in {csv_root}.")
        return False

def stat(EAResultRoot: Path):
    """
        count escape reason according to reasons defined in escape/stat_subgraph.go

        read from EAResultRoot/all.csv
        
        The header is:
        nodeid,reason,pkg
        
        column  dtype
        reason  define.EscapeReason
        nodeid  int
        pkg     string

        1. count the number and percentile of nodes for each reason
        2. count the distribution of number of reasons each node have
        3. Draw a DAG. Each node is a group of reasons. Parent node's reasons contain children's reasons
    """
    print(f"Stat {EAResultRoot} ...")
    log_path = EAResultRoot / "escape_reason_stat.txt"
    with open(log_path, "w", encoding="utf-8") as f, redirect_stdout(f):
        df = pd.read_csv(EAResultRoot / "all.csv")

        # Convert numeric reason to EscapeReason enum object
        df["reason_enum"] = df["reason"].apply(lambda val: EscapeReason(val))

        # 1. Count frequency of each individual reason
        from collections import Counter
        all_reasons = []
        for reasons in df["reason_enum"]:
            all_reasons.extend(EscapeReason(reasons).list_reasons())

        reason_counter = Counter(all_reasons)
        total_nodes = len(df)
        
        print("[1] Frequency and percentage of each reason:")
        for reason, count in reason_counter.items():
            percent = count / total_nodes * 100
            print(f"{reason.name:<20} {count:>5} ({percent:.2f}%)")

        # 2. Distribution of number of reasons per node
        df["num_reasons"] = df["reason_enum"].apply(lambda r: len(EscapeReason(r).list_reasons()))
        num_reason_dist = df["num_reasons"].value_counts().sort_index()

        print("\n[2] Distribution of number of reasons per node:")
        for num, count in num_reason_dist.items():
            print(f"{num} reasons: {count} nodes")

        # 3. Directed DAG: reason combinations with subset relation
        # group by sorted reason name tuple
        def reasons_key(val: int):
            return tuple(sorted(r.name for r in EscapeReason(val).list_reasons()))

        grouped = df.groupby(df["reason"].apply(reasons_key))
        reason_groups = [(set(key), len(group)) for key, group in grouped]

        # 构建 DAG：每个节点是一个 reason 组合（用 tuple 表示），边表示严格包含关系
        G = nx.DiGraph()

        # 添加节点
        for key_set, count in reason_groups:
            key = tuple(sorted(key_set))  # node identifier: sorted tuple
            label = "+".join(key) + f" ({count})"
            G.add_node(key, label=label, reasons=key_set, count=count)

        # 添加边（如果 A 是 B 的真超集，则 B -> A）
        for i, (reasons_a, _) in enumerate(reason_groups):
            for j, (reasons_b, _) in enumerate(reason_groups):
                if i == j:
                    continue
                if reasons_a > reasons_b:  # A ⊃ B
                    key_a = tuple(sorted(reasons_a))
                    key_b = tuple(sorted(reasons_b))
                    G.add_edge(key_b, key_a)  # edge B → A


        # 可视化
        print("\n[3] Reason DAG (subset relationships):")
        pos = nx.spring_layout(G, seed=42, k=1.5)
        labels = {node: G.nodes[node]["label"] for node in G.nodes}
        nx.draw(G, pos, with_labels=True, labels=labels, node_size=1500, node_color='lightblue', font_size=8, edge_color='gray')
        plt.title("EscapeReason DAG")
        plt.tight_layout()

        output_path = EAResultRoot / "escape_reason_dag.png"
        plt.savefig(output_path, dpi=200)
        plt.close()
    
def run(repoRoot: Path, statRoot: Path, repo: str, repoBuild: Callable[[Path, Path], None]=build):
    repoPath = repoRoot/repo
    EAResultRoot = statRoot/repo
    EAResultRoot.mkdir(parents=True, exist_ok=True)
    if do_build:
        build(repoPath, EAResultRoot)
    has_csv = collect(EAResultRoot)
    if has_csv:
        stat(EAResultRoot)

def demo():
    for repo in [
        # "closure_go",
        # "indirect",
        # "paramleak",
        # "ret",
        # "rootstack",
        "easyPkg",
    ]:
        run(Path("."), Path.cwd()/"stat_root", repo)

def cli_build(repoPath: Path, EAResultRoot: Path):
    execute("make clean", cwd=str(repoPath))
    execute("make install", cwd=str(repoPath))

def tidb_build(repoPath: Path, EAResultRoot: Path):
    execute("make clean", cwd=str(repoPath))
    execute("make", cwd=str(repoPath))

def grpc_go_build(repoPath: Path, EAResultRoot: Path):
    execute("make deps", cwd=str(repoPath))
    execute("make build", cwd=str(repoPath))

def rclone_build(repoPath: Path, EAResultRoot: Path):
    execute(f'make BUILD_ARG=\'-a -gcflags="all=-eareasonstat -earesultroot={EAResultRoot}"\'', cwd=str(repoPath))
    
def escapeReasonRepos():
    for repo, repobuild in [
        # ("cli", cli_build),
        # ("tidb", tidb_build),
        # ("hugo", build),
        # ("gogs", build),
        # ("grpc-go", grpc_go_build),
        # ("rclone", rclone_build)

        # build method unknown
        ("sonic", build),
        ("fastgo", build), # fail
        ("hertz", build),
        ("hertz/cmd/hz", build),
        ("hertz/internal/bytesconv", build),
        ("hertz/internal/bytestr", build),
        ("kitex", build),
        ("kitex/client", build),
        ("kitex/server", build),
    ]:
        run(
            Path("/data/github_go/escapeReasonRepos"),
            Path("/data/github_go/escapeReasonStatRoot"),
            repo,
            repobuild
        )

def main():
    if demo_only:
        demo()
    else:
        escapeReasonRepos()

parser = argparse.ArgumentParser(prog="run")
parser.add_argument("--build", action="store_true", help="Whether to build the project")
parser.add_argument("--demo", action="store_true", help="Whether to run demo only")
args = parser.parse_args()
do_build = args.build
demo_only = args.demo
if __name__ == "__main__":
    main()