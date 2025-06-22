import subprocess
from pathlib import Path
from typing import List
import pandas as pd

dry_run = False

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
    execute(f'go build -a -gcflags="-m=2 -earesultroot={EAResultRoot}" .', cwd=str(repo))

def collect(EAResultRoot: Path):
    """ Collect all all.csv from EAResultRoot/csv.
        The result is written under EAResultRoot/csv.

        For each package import path, create a new DataFrame by adding "pkg" column in DataFrame of pkg/import/path/all.csv using the relative path to EAResultRoot and dump the collected all packages' DataFrame in EAResultRoot/csv/all.csv

    Args:
        root (Path): EAResultRoot
    """
    csv_root = EAResultRoot.joinpath("csv")
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
        all_df.to_csv(EAResultRoot.joinpath("all.csv"), index=False)
    else:
        print(f"No all.csv files found in {csv_root}.")

def stat(EAResultRoot: Path):
    pass
    

def main():
    repos = [Path(s) for s in [
        "closure_go",
        "indirect",
        "paramleak",
        "ret",
        "rootstack"
    ]]
    for repo in repos:
        EAResultRoot = Path.cwd().joinpath("stat_root", repo)
        EAResultRoot.mkdir(parents=True, exist_ok=True)
        build(repo, EAResultRoot)
        collect(EAResultRoot)

if __name__ == '__main__':
    main()

