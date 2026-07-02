%
% Canned GML example scene
%

% color refl fuzz transparency refr kd ks n material

% Glass sphere with metallic sheen

{
    /v /u /face		  % bind arguments
    0.8 0.2 0.2 point % color
    0.0               % reflectivity
    0.0               % fuzz
    0.9               % transparency
    1.5               % refractive index
    1.0               % kd
    0.8               % ks
    50.0              % specular exponent (n)
    material
}
sphere
0.0 0.0 5.0 translate
/glassSphere

% Dull fuzzy sphere

{
	/v /u /face		  % bind arguments
    0.2 0.2 0.8 point % color
    0.2               % reflectivity
    0.5               % fuzz
    0.0               % transparency
    0.0               % refractive index
    1.0               % kd
    0.0               % ks
    0.0               % specular exponent (n)
    material
}
sphere
2.0 0.0 8.0 translate
/dullSphere

% Reflective green sphere

{
	/v /u /face		  % bind arguments
    0.2 0.8 0.2 point % color
    0.8               % reflectivity
    0.0               % fuzz
    0.0               % transparency
    0.0               % refractive index
    1.0               % kd
    0.0               % ks
    0.0               % specular exponent (n)
    material
}
sphere
-2.0 0.0 6.0 translate
/greenSphere

% Ground plane
% We use a giant far away sphere for the ground plane because reasons.
{ /v /u /face
  0.8 0.8 0.8 point
  1.0 0.0 0.0
} sphere
0.0 -1001.0 5.0 translate
1000.0 uscale
/groundPlane

groundPlane
glassSphere  union
dullSphere   union
greenSphere  union
/scene

% Lights

5.0 5.0 0.0 point
1.0 1.0 1.0 point pointlight /light

0.1 0.1 0.1 point		      % ambient light
[ light ]				      % lights
scene				          % scene to render
7				              % tracing depth
120.0				          % field of view
1900 1200 		              % image width and height
"canned.ppm"			      % output file
0.0 0.0 0.0 point             % bg start
0.5 0.7 1.0 point             % bg end
renderWithBgGradient
