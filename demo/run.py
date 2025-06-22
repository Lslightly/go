import subprocess
from typing import List

def execute(cmd: str, cwd: str | None = None, stdout=None, stderr=None):
    proc = subprocess.run(cmd,
            shell=True, text=True,
            cwd=cwd,
            stdout=stdout, stderr=stderr)
    return proc.returncode, proc.stdout, proc.stderr

def build(dirs: List[str]):
    for dir in dirs:
        print(f"Building {dir}...")
        execute('go build -a -gcflags="all=-m=2" .', cwd=dir)

def main():
    dirs = [
        "closure_go",
        "indirect",
        "paramleak",
        "ret",
        "rootstack"
    ]
    build(dirs)

if __name__ == '__main__':
    main()

