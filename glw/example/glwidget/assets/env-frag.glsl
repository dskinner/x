#version 100
#define onesqrt2 0.70710678118
#define sqrt2 1.41421356237
#define pi 3.14159265359
#define twopi 6.28318530718
precision mediump float;

// radius values in 0..1 clamp border;
// zero is rectangle, one is ellipse,
// and less than one is rounded corners.
uniform float radius;

// color background.
uniform vec4 color;

// content to display; required.
uniform sampler2D content;

// fragST is content coordinate to map to fragPos.
varying vec2 fragST;

varying vec4 fragPos;

// clampRadius calls discard if fragPos is outside radius.
void clampRadius() {
  // distance from fragPos to center; given by mapping -1..1 to 0..1..0
  vec2 dist = 1.0-abs(fragPos.xy);
  // clipping distance components to <=radius allows for rounded corners.
  if (dist.x <= radius && dist.y <= radius) {
    float d = length(1.0-(dist/radius));
    if (d > 1.0) {
      discard;
    }
  }
}

void main() {
  vec4 c = texture2D(content, fragST.xy);
  gl_FragColor = mix(color, c, c.a);

  clampRadius();
}
