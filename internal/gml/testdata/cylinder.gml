% sylinder.gml
%
% OUTPUTS: cylinder0.ppm cylinder1.ppm cylinder2.ppm cylinder3.ppm
%
% test cylinder geometry and basic texturing
%

#include "colors.ins"

{ /v /u /face
  face 0 eqi
  { 0.3 0.3 u point }
  { face 1 eqi { red } { green } if }
  if
  1.0 0.0 1.0
} cylinder
  0.0 0.0 3.0 translate /box

% Transforms compose so that the most recently applied one acts on the
% object first (closest to its own local origin), with earlier ones acting
% afterward, outermost. So `box`'s "move away from camera" translate above
% is written first, making it the outermost (last-applied) transform: it
% always pushes the current view's fully-rotated cylinder out to Z=3,
% regardless of what per-view rotation doit applies below. doit's own
% translate below then runs innermost, centering the raw cylinder before
% any per-view rotation, so rotations pivot around the cylinder's own
% center rather than swinging the whole object off-frame.
{ /file /box
  1.0 1.0 1.0 point
  []
  box 0.0 -0.5 0.0 translate
  1
  90.0
  320 200
  file
  render
} /doit

% render front view
box "cylinder0.ppm" doit apply

% render bottom view
box 90.0 rotatex "cylinder1.ppm" doit apply

% render top view
box -90.0 rotatex "cylinder2.ppm" doit apply

% render back view
box 180.0 rotatey "cylinder3.ppm" doit apply

