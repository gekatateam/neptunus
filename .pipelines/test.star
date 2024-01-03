load("math.star", "math")

def apply(event):
    event.setField("from_test", math.sqrt(1337))

test = struct(
    apply = apply
)
