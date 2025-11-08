import argparse
import functools
import subprocess
from pathlib import Path
import tempfile
from typing import Callable, List, Dict
from define import EscapeReason
import networkx as nx
import matplotlib.pyplot as plt
from contextlib import redirect_stdout
import os
import shutil
from tqdm import tqdm

import pandas as pd

dry_run = False
do_build = False   # build repo
do_collect = False # collect csv
do_stat = False    # stat csv
demo_only = False  # only run demo

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
        pkg_path = csv_path.parent.parent.relative_to(csv_root)
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
        
        print(f"Number of nodes: {total_nodes}")
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

def execute(cmd: str, cwd: str | None = None, stdout=None, stderr=None, extra_env=None, tqdmlog=False):
    if dry_run:
        print("pwd:", cwd, "cmd:`", cmd, '`')
        return 0, "", ""
    final_env = os.environ.copy()
    if extra_env is not None:
        final_env.update(extra_env)
    if tqdmlog:
        tqdm.write(f"cmd: {cmd}, cwd: {cwd}, extra_env: {extra_env}")
    proc = subprocess.run(cmd,
            shell=True, text=True,
            cwd=cwd,
            stdout=stdout, stderr=stderr,
            env=final_env)
    return proc.returncode, proc.stdout, proc.stderr

def gen_extra_env(EAResultRoot: Path) -> Dict:
    return {
        "EAReasonStat": "1",
        "EAResultRoot": str(EAResultRoot),
    }
    

def build(repo: Path, repoBuildLogRoot: Path, EAResultRoot: Path, cmd: str='go build -a .', ESGDumpEnabled: bool=False):
    """ Use 'go build -a .' command
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
                    ...
    Args:
        dir (str): source dir
        EAResultRoot (str): root of escape analysis result
    """
    print(f"Building {repo}...")
    extra_env = gen_extra_env(EAResultRoot)
    if ESGDumpEnabled:
        extra_env["ESGDumpEnabled"] = "1"
    print(f"Cmd: {cmd}")
    with open(repoBuildLogRoot/"stdout", 'w') as stdout, open(repoBuildLogRoot/"stderr", 'w') as stderr:
        execute(
            cmd,
            cwd=str(repo),
            extra_env=extra_env,
            stdout=stdout,
            stderr=stderr,
        )

class GenStdLibException(Exception):
    
    def __init__(self):
        super().__init__("Generate stdlib importcfg failed")
    
    def __str__(self):
        return "Generate stdlib importcfg failed"

def genStdlibImportcfg() -> str:
    ret, out, _ = execute(
        cmd="go list -export -f '{{if .Export}}packagefile {{.ImportPath}}={{.Export}}{{end}}' std",
        extra_env={
            'GOENV': 'off',
            'GOFLAGS': '',
        },
        stdout=subprocess.PIPE,
    )
    if ret != 0:
        raise GenStdLibException
    return out

def escape2_build(repoPath: Path, repoBuildLogRoot: Path, EAResultRoot: Path):
    import_cfg_path = Path(tempfile.gettempdir())/"importcfg"
    with open(import_cfg_path, 'w') as import_cfg:
        import_cfg.write(genStdlibImportcfg())
    build(repoPath, repoBuildLogRoot, EAResultRoot, cmd=f"go tool compile -d=panic -p=p -C -e -importcfg={import_cfg_path} -o a.o -l -d=ssa/check/on escape2.go", ESGDumpEnabled=True)

def cli_build(repoPath: Path, repoBuildLogRoot: Path, EAResultRoot: Path):
    with open(repoBuildLogRoot/"stdout", 'w') as stdout, open(repoBuildLogRoot/"stderr", 'w') as stderr:
        execute("make clean", cwd=str(repoPath), stdout=stdout, stderr=stderr)
        print('make clean finish.', file=stdout)
    with open(repoBuildLogRoot/"stdout", 'a') as stdout, open(repoBuildLogRoot/"stderr", 'a') as stderr:
        execute("make", cwd=str(repoPath), extra_env=gen_extra_env(EAResultRoot), stdout=stdout, stderr=stderr)
        print('make finish.', file=stdout)

def get_go_path():
    _, path, _ = execute("go env GOPATH", stdout=subprocess.PIPE)
    return path.strip()

def tidb_build(repoPath: Path, repoBuildLogRoot: Path, EAResultRoot: Path):
    """
    all: dev server benchkv

    go build -a
    go test -a
    go install -a

    dev:
        integrationtest -> server_check:
            go build
        br_unit_test
            go test
        parser_unit_test
            go test
        ut -> tools/bin/ut
            go build
        ut -> tools/bin/xprog
            go build
        uv -> tools/bin/failpoint-ctl:
            go install
    
    server:
        go build
    
    benchkv:
        go build
    """
    common_env = {
        "GOPATH": get_go_path(),
    }
    tgts = [
        "server_check",
        "br_unit_test",
        "parser_unit_test",
        "tools/bin/ut",
        "tools/bin/xprog",
        "tools/bin/failpoint-ctl",
        "server",
        "benchkv",
    ]
    tgt2root_dir = {
        tgt: EAResultRoot/tgt
        for tgt in tgts
    }
    for subdir in tgt2root_dir.values():
        subdir.mkdir(parents=True, exist_ok=True)
    with open(repoBuildLogRoot/"stdout", 'w') as stdout, open(repoBuildLogRoot/"stderr", 'w') as stderr:
        execute("make clean", cwd=str(repoPath), extra_env=common_env, stdout=stdout, stderr=stderr)
        for tgt, root_dir in tgt2root_dir.items():
            extra_env = gen_extra_env(root_dir)
            extra_env.update(common_env)
            execute(f"make {tgt}", cwd=str(repoPath), extra_env=extra_env, stdout=stdout, stderr=stderr, tqdmlog=True)


def grpc_go_build(repoPath: Path, repoBuildLogRoot: Path, EAResultRoot: Path):
    """
    build
        go build
    test:
        go test
    testsubmodule:
        go test
    """
    with open(repoBuildLogRoot/"stdout", 'w') as stdout, open(repoBuildLogRoot/"stderr", 'w') as stderr:
        # execute("make deps", cwd=str(repoPath), stdout=stdout, stderr=stderr)
        execute("make build", cwd=str(repoPath), extra_env=gen_extra_env(EAResultRoot), stdout=stdout, stderr=stderr)

def rclone_build(repoPath: Path, repoBuildLogRoot: Path, EAResultRoot: Path):
    with open(repoBuildLogRoot/"stdout", 'w') as stdout, open(repoBuildLogRoot/"stderr", 'w') as stderr:
        execute('make', cwd=str(repoPath), extra_env=gen_extra_env(EAResultRoot), stdout=stdout, stderr=stderr)

def run(repoRoot: Path, buildLogRoot: Path, statRoot: Path, repo: str, repoBuild: Callable[[Path, Path, Path], None]=build):
    repoPath = repoRoot/repo
    EAResultRoot = statRoot/repo
    repoBuildLogRoot = buildLogRoot/repo
    if do_build:
        if EAResultRoot.exists():
            shutil.rmtree(EAResultRoot)
        EAResultRoot.mkdir(parents=True, exist_ok=True)
        if repoBuildLogRoot.exists():
            shutil.rmtree(repoBuildLogRoot)
        repoBuildLogRoot.mkdir(parents=True, exist_ok=True)
        repoBuild(repoPath, repoBuildLogRoot, EAResultRoot)
    has_all_csv = False
    if do_collect:
        has_all_csv = collect(EAResultRoot)
    if do_stat:
        if has_all_csv:
            stat(EAResultRoot)
        else:
            print(f"does not have all.csv for {str(repo)} but do stat")

def demo():
    for repo in [
        "closure_go",
        "go_interprocedural",
        "indirect",
        "paramleak",
        "ret",
        "rootstack",
        "easyPkg",
        "escape",
        "boc",
        "printf",
        "closure_return",
        "global",
    ]:
        build_with_esg_dumping = functools.partial(build, ESGDumpEnabled=True)
        run(Path.cwd()/"fixtures", Path.cwd()/"build_log", Path.cwd()/"stat_root", repo, build_with_esg_dumping)
    for repo, repobuild in [
        ("escape2", escape2_build)
    ]:
        run(Path.cwd()/"fixtures", Path.cwd()/"build_log", Path.cwd()/"stat_root", repo, repobuild)
    
def escapeReasonRepos():
    for repo, repobuild in tqdm([
        ("fzf", build),
        ("cli", cli_build),
        # ("tidb", tidb_build),
        ("hugo", build),
        ("gogs", build),
        ("grpc-go", grpc_go_build),
        # ("rclone", rclone_build),

        # # build method unknown
        # ("sonic", build),
        # ("fastgo", build), # fail
        # ("hertz", build),
        # ("hertz/cmd/hz", build),
        # ("hertz/internal/bytesconv", build),
        # ("hertz/internal/bytestr", build),
        # ("kitex", build),
        # ("kitex/client", build),
        # ("kitex/server", build),
    ], desc="building escape reason repos"):
        run(
            Path("/data/github_go/escapeReasonRepos"),
            Path("/data/github_go/escapeReasonBuildLog"),
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
parser.add_argument("--demo", action="store_true", help="Whether to run demo only")
parser.add_argument("--build", action="store_true", help="Whether to build the project")
parser.add_argument("--collect", action="store_true", help="Whether to collect all.csv")
parser.add_argument("--stat", action="store_true", help="Whether to stat based on all.csv")
args = parser.parse_args()
demo_only = args.demo
do_build = args.build
do_collect = args.collect
do_stat = args.stat
if __name__ == "__main__":
    main()
