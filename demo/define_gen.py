fields = [
    ("ERHeap", "root is heap", "h"),
    ("ERStar", "root is *x", "*X"),
    ("ERRootStack", "root in stack", "rs"),
    ("EROutliveRootStack", "outlived by root in stack", "ors"),
    ("ERClosure", "root is closure", "clo"),
    ("ERGo", "root is go", "go"),
    ("ERDefer", "root is defer", "defer"),
    ("ERReturn", "l is returned", "ret"),
    ("ERParamLeak", "param leak", "param"),
    ("ERMapIdx", "m[index]", "mapidx"),
    ("ERSliceIdx", "s[index]", "sliceidx"),
    ("ERComm", "communication by send", "send"),
    ("ERAppend", "append", "append"),
    ("ERCopy", "copied slice", "copy"),
    ("ERMust", "must be heap allocated", "must"),
    ("ERNumber", "marker", None),
]

print("class EscapeReason(IntFlag):")
for i, (name, doc, _) in enumerate(fields):
    print(f"    {name} = 1 << {i:<2}  # {doc}")

print("\nreason_to_xlabel = {")
for name, _, label in fields:
    if label:
        print(f"    EscapeReason.{name}: \"{label}\",")
print("}")
