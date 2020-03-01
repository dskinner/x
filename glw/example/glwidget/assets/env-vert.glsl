#version 100

uniform mat4 proj;
uniform mat4 model;

attribute vec4 vertex;
attribute vec2 contentCoord;

varying vec4 fragPos;
varying vec2 fragST;

void main() {
  gl_Position = proj*model*vertex;
  fragPos = vertex;
  fragST = contentCoord;
}
