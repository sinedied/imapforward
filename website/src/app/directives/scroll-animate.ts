import {Directive, ElementRef, afterNextRender, input} from '@angular/core';

@Directive({
  selector: '[appScrollAnimate]',
})
export class ScrollAnimate {
  readonly threshold = input(0.15);

  constructor(private readonly el: ElementRef<HTMLElement>) {
    this.el.nativeElement.classList.add('scroll-animate');

    afterNextRender(() => {
      const observer = new IntersectionObserver(
        ([entry]) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('visible');
            observer.unobserve(entry.target);
          }
        },
        {threshold: this.threshold()},
      );
      observer.observe(this.el.nativeElement);
    });
  }
}
