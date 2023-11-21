(use-modules (ice-9 format)
             (ice-9 match)
             (srfi srfi-43)
             (srfi srfi-64)
             (fibers)
             (fibers channels)
             (fibers conditions)
             (fibers operations)
             (dice))

(test-begin "tests")

;; all vector elements are unique
(let ((v (gen 3 4)))
  (vector-for-each
   (lambda (_ a)
     (test-eq 1 (vector-count (lambda (_ b) (equal? b a)) v)))
   v))

;; action equality
(let ((a (make-action 1 3))
      (b (make-action 1 3))
      (c (make-action 2 3)))
  (test-assert (equal? a b))
  (test-assert (not (equal? a c))))

;; by action speed, then power
(let ((a (make-action 6 2))
      (b (make-action 6 3))
      (c (make-action 1 4))
      (d (make-action 2 4))
      (e (make-action 2 4)))
  (test-assert (less-speed-power a b))
  (test-assert ((negate less-speed-power) b a))
  
  (test-assert (less-speed-power b c))
  (test-assert ((negate less-speed-power) c b))
  
  (test-assert (less-speed-power c d))
  (test-assert ((negate less-speed-power) d c))
  
  (test-assert ((negate less-speed-power) d e))
  (test-assert ((negate less-speed-power) e d)))

;; action-list-penalize
;; TODO

;; conflict resolution
(let ((a '())
      (b '()))
  (test-equal (values 0 0) (conflict-resolve a 0 b 0)))

(let ((a (list (make-action 6 2)))
      (b '()))
  (test-equal (values 1 0) (conflict-resolve a 0 b 0)))

(let ((a '())
      (b (list (make-action 6 2))))
  (test-equal (values 0 1) (conflict-resolve a 0 b 0)))

(let ((a (list (make-action 6 2)))
      (b (list (make-action 6 2))))
  (test-equal (values 0 0) (conflict-resolve a 0 b 0)))

(let ((a (list (make-action 5 2)))
      (b (list (make-action 6 2))))
  (test-equal (values 0 1) (conflict-resolve a 0 b 0)))

(let ((a (list (make-action 6 2)))
      (b (list (make-action 5 2))))
  (test-equal (values 1 0) (conflict-resolve a 0 b 0)))

(let ((a (list (make-action 5 3)))
      (b (list (make-action 6 2))))
  (test-equal (values 1 0) (conflict-resolve a 0 b 0)))

(let ((a (list (make-action 5 3)))
      (b (list (make-action 6 3))))
  (test-equal (values 1 1) (conflict-resolve a 0 b 0)))

(let ((a (gen-actions-valid-sorted 3 6))
      (b (gen-actions-valid-sorted 4 6)))
  (test-assert (display (format #f "3v3d6 ~a\n" (result->string (resolve-all a a)))))
  (test-assert (display (format #f "3v4d6 ~a\n" (result->string (resolve-all a b)))))
  (test-assert (display (format #f "4v4d6 ~a\n" (result->string (resolve-all b b))))))

;; test fibers

(run-fibers
 (lambda () 
   (let* ((a (gen-actions-valid-sorted 3 6))
          (b (gen-actions-valid-sorted 4 6))
          (r (resolve-all a b))
          (s (resolve-all-split a b))
          (t (resolve-all-fan-out a b)))
     (test-assert (equal? r s))
     (test-assert (equal? r t))))
 #:drain? #t)

(test-end "tests")

(define-syntax-rule (benchmark e ...)
  (let ((now (tms:clock (times))))
    (begin e ...)
    (display (format #f "~,2fs ~a\n" (/ (- (tms:clock (times)) now) 1e9) (quote e ...)))))

(define (-bench)
  ;; 0.05s
  (benchmark (gen 6 6))
  ;; 0.12s
  (benchmark (gen-actions 6 6))
  ;; 0.18s
  (benchmark (gen-actions-valid-sorted 6 6))
  ;; 0.28s
  (benchmark (resolve-all (gen-actions-valid-sorted 4 6) (gen-actions-valid-sorted 4 6)))

  ;; 2.01s
  (benchmark (resolve-all (gen-actions-valid-sorted 4 6) (gen-actions-valid-sorted 5 6)))
  ;; 0.65s
  (benchmark (resolve-all-fan-out (gen-actions-valid-sorted 4 6) (gen-actions-valid-sorted 5 6)))

  ;; 16.67s
  ;; (benchmark (resolve-all (gen-actions-valid-sorted 5 6) (gen-actions-valid-sorted 5 6)))
  ;; 5.18s
  ;; (benchmark (resolve-all-fan-out (gen-actions-valid-sorted 5 6) (gen-actions-valid-sorted 5 6)))

  ;; 42.56s
  ;; (benchmark (resolve-all-fan-out (gen-actions-valid-sorted 5 6) (gen-actions-valid-sorted 6 6)))

  ;; 335.60s
  ;; (benchmark (resolve-all-fan-out (gen-actions-valid-sorted 6 6) (gen-actions-valid-sorted 6 6)))

  )

(define (bench args)
  (display (format #f "%%%% Starting benchmarks ~a\n" args))
  (run-fibers -bench #:drain? #t))

(define (plot-vs n0 n1)
  ;; sorting by wins illustrates the small number of groups all rolls will fall into
  ;; TODO check if tie,loss,part are all equal within each group
  (let* ((a (gen-actions-valid-sorted n0 6))
         (b (gen-actions-valid-sorted n1 6))
         (rs (list->vector (sort (resolve-each-fan-out a b) less-win))))
    (with-output-to-file (format #f "n~av~ad6.dat" n0 n1)
      (lambda ()
        (vector-for-each
         (lambda (i r)
           (display (format #f "~f ~f\n" (/ (+ 1 i) (vector-length rs)) (result-win-percent r))))
         rs)))))

(define (-plot)
  (plot-vs 4 3)
  (plot-vs 4 4)
  (plot-vs 4 5))

(define (plot args)
  (display (format #f "%%%% Starting plots ~a\n" args))
  (run-fibers -plot #:drain? #t))
