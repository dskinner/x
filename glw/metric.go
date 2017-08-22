package glw

import "errors"

/*
First I found a list of transforms in the field of mathematics on wikipedia:
https://en.wikipedia.org/wiki/List_of_transforms

Then I found Sequential euclidean distance transforms:
https://en.wikipedia.org/wiki/Sequential_euclidean_distance_transforms

This led me to the Euclidean distance or Euclidean metric:
https://en.wikipedia.org/wiki/Euclidean_distance

Currently I have Scale. I've been anticipating this becoming Size or a similar name. Currently it's Scale b/c of the projection matrix where screen height is (-1 .. +1) and aspect ratio is maintained for width. This made the imageviewer easier to implement instead of dealing with pixels.

But this will no longer be relevant with constraint based layout so Scale no longer makes sense except as a function that acts on Size.

Both fields then, Translate and Size, are euclidean distances. Size draws a line perpendicular to its boundaries. Translate draws a line from an origin point, normally (0, 0).

Field Rotate is a quaternion, describing a distance along a circle.

Ok, but "distance or metric"? What's a metric? Which led me to:
https://en.wikipedia.org/wiki/Metric_(mathematics)

A metric is a distance function. A few moments before the start of finding any of this, I was considering the performance impact of generating a stack of closures that are evaluated during an animation sequence or a layout phase that calculates any missing values.

So, a product of the constraint-based layout will be a set of metrics "that defines a distance between each pair of elements of a set". Such a set is called a metric space:
https://en.wikipedia.org/wiki/Metric_space

"A metric space is a set for which distances between all members of the set are defined". I do not know what the inverse of this is called, but such is what the constraint-based layout will operate on using it's first product, the metric space produced from programmer input.

Once all undefined members values are defined, the layout process is complete and has produced a final metric space; that is, a set of metrics.

This set of metrics is what enforces conformance of other objects when one object animates.

And that set of metrics could very well be a slice of closures. It's certainly be the simplest implementation. Not writing any code until I develop a full picture of all interfaces.

---

One thing I just realized is that during iteration, not all incomplete metrics can be evaluated right away; they must be weighted by importance.

If there exists a metric where the right of the left-box and the left of the right-box are undefined, this can evaluate so distance reaches zero and they meet center of parent.

But if there follows another metric for evaluation that assigns a discrete value to the left of the right-box, this would alter the evaluation of the previous metric.

So, discrete value assigns have a higher weight than undefines. Each iteration needs to sort the incomplete metrics based on dynamically calculated weight given that each iteration step generates discrete value assignments for values previously not defined.

Sort order can be maintained by pop, and insert methods operating on a binary search tree of the set.

I'm defining a new term, opaque metric: both values for calculating distance are undefined.

Opaque metrics may come in any order from a programmer. All weighted the same, this means for a given set of metrics, sort is not stable. Each iteration assigns discrete values based on sort order. This means the end evaluation for a given set depends on the sorted order. This means sort should be made stable for predictable results.

Right now I'm just thinking priority goes to boundary location; left, top, right, bottom.

---

I've been reading about closed sets and set closures:
https://en.wikipedia.org/wiki/Closed_set
https://en.wikipedia.org/wiki/Closure_(mathematics)

Under examples of closed sets, I came across this:
"The Cantor set is an unusual closed set in the sense that it consists entirely of boundary points and is nowhere dense."
https://en.wikipedia.org/wiki/Cantor_set

That sounds a lot like what I'm doing.

I also found this example illuminating:
"The unit interval [0,1] is closed in the metric space of real numbers"

What's interesting about that is the "the metric space of real numbers" makes clear to me how to go from programmer defined metrics to data storage. Compared to package material which used struct hierarchies that were ignorant of siblings right up until trying to resolve layout, this is telling me that each such input can go directly into the set when created.

Why is that interesting? As I was reading about closed sets, I found that a closed set can be formally verified as closed. Initially I thought that would be excellent for writing tests, but if the verification is cheap then problems can be identified much early in the process.

What I'm trying to address now is what the data type of a metric would actually look like as code. I'll be reading about cantor sets.

---

"the most common modern construction is the Cantor ternary set, built by removing the middle thirds of a line segment"

I was already thinking a metric would look like defining a ternary relation, I think I'm on the right track.

"Cantor himself mentioned the ternary construction only in passing, as an example of a more general idea, that of a perfect set that is nowhere dense."

This scares me, a perfect set that is nowhere dense. When I was in AR, I told about the perfect solution to a problematic is one that produces no side effects. That's so perfectly theoretical it's perfectly useless as the point of any software project is to produce side effects. The reason for considering the perfect solution is to stay grounded in the number of side effects your generating, and to reduce as much as possible to nil. I think my perfect solution rings of thinking like a mathematician.
*/

/*
think about as a function of sound; the left input and right input collide to the center; SetLeftInput, SetRightInput, LeftOutput, RightOutput; this is 1 dimensional; consider pkg snd; represented as a linked-list:

var frame, lbl, btn T
frame.SetSize(0, 0, 800, 600)
lbl.LeftInput = frame.LeftOutput
btn.LeftInput = lbl.RightOutput
btn.RightInput = frame.RightOutput

lbl.LeftInput == frame.LeftOutput
lbl.RightInput == btn.LeftInput // ??? btn.SetLeftInput(lbl) or (lbl.RightOutput) which still allows linking to lbl ??? (lbl, RightOutput)

frame.SetBounds(0, 0, 800, 600)
lbl.StartAt(frame, Start)
btn.StartAt(lbl, End)
btn.EndAt(frame, End)

*/

type Side int

const (
	Start Side = iota
	End
	Top
	Bottom
)

type T struct {
}

func (a *T) SetWidth(x int) {}

func (a *T) SetHeight(x int) {}

func (a *T) StartAt(t T, d Side) {}

func (a *T) EndAt(t T, d Side) {}

func (a *T) TopAt(t T, d Side) {}

func (a *T) BottomAt(t T, d Side) {}

var ErrNotZero = errors.New("Metric distance not zero")

// cset represents a cantor set; iterations are applied by bitshifting for
// powers of two with the result being a count of segments.
// type cset uint64

type metric struct {
	w, h int
}

func (a metric) Do() error { return ErrNotZero }

type metricSpace []metric

func (a metricSpace) Closed() bool { return false }
