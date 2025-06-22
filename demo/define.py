from enum import IntFlag, auto, EnumMeta

# 自定义 metaclass，支持 1 << n 自增行为
class BitShiftEnumMeta(EnumMeta):
    def __new__(metacls, cls, bases, classdict):
        value = 1
        for key in classdict._member_names:
            if classdict[key] == auto():
                classdict[key] = value
                value <<= 1
            else:
                value = classdict[key] << 1
        return super().__new__(metacls, cls, bases, classdict)

class EscapeReason(IntFlag, metaclass=BitShiftEnumMeta):
    ERHeap = auto()
    ERStar = auto()
    ERRootStack = auto()
    EROutliveRootStack = auto()
    ERClosure = auto()
    ERGo = auto()
    ERDefer = auto()
    ERReturn = auto()
    ERParamLeak = auto()
    ERMapIdx = auto()
    ERSliceIdx = auto()
    ERComm = auto()
    ERAppend = auto()
    ERCopy = auto()
    ERMust = auto()
    ERNumber = auto()

    def list_reasons(self):
        """Return list of individual EscapeReason flags in this value (excluding ERNumber)."""
        return [reason for reason in EscapeReason if reason != EscapeReason.ERNumber and reason in self]

    def labels(self):
        """Return list of short labels corresponding to the flags in this value."""
        return [reason_to_xlabel[r] for r in self.list_reasons()]

# 标签映射表
reason_to_xlabel = {
    EscapeReason.ERHeap: "h",
    EscapeReason.ERStar: "*X",
    EscapeReason.ERRootStack: "rs",
    EscapeReason.EROutliveRootStack: "ors",
    EscapeReason.ERClosure: "clo",
    EscapeReason.ERGo: "go",
    EscapeReason.ERDefer: "defer",
    EscapeReason.ERReturn: "ret",
    EscapeReason.ERParamLeak: "param",
    EscapeReason.ERMapIdx: "mapidx",
    EscapeReason.ERSliceIdx: "sliceidx",
    EscapeReason.ERComm: "send",
    EscapeReason.ERAppend: "append",
    EscapeReason.ERCopy: "copy",
    EscapeReason.ERMust: "must",
}