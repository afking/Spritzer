Spritzer
========

Simple sprite packager

How
---

Spritzer collects all png image files and combines them into a sprite sheet. 

- Normal : "Name.png" 
- Retina : "NameRetina.png"
- Output : "sprite.png", "spriteRetina.png" and "sprite.css"

It uses the suffix "Retina" to destinguish pngs of double size which it will use for a sprite sheet on higher dpi displays.
Spritzer also outputs a basic css for background positions.
