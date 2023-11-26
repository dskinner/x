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
   (λ (_ a)
     (test-eq 1 (vector-count (λ (_ b) (equal? b a)) v)))
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

;; penalize-action-list
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

;;
(let ((a (gen-actions 3 6))
      (b (gen-actions 4 6)))
  (test-assert (format #t "3v3d6 ~a\n" (result->string (resolve-all a a) #:percent? #t)))
  (test-assert (format #t "3v4d6 ~a\n" (result->string (resolve-all a b) #:percent? #t)))
  (test-assert (format #t "4v4d6 ~a\n" (result->string (resolve-all b b) #:percent? #t))))

;; (let ((a (gen-actions 4 6))
;;       (b (gen-actions 5 6))
;;       (c (gen-actions 4 6 mod-1g!))
;;       (d (gen-actions 4 6 mod-2432!))
;;       (e (gen-actions 4 6 mod-2432! mod-1g!)))
  
;;   (format #t "4v5d6 ~a\n" (result->string (resolve-all a b) #:percent? #t))
;;   (format #t "1g 4v5d6 ~a\n" (result->string (resolve-all c b) #:percent? #t))
;;   (format #t "2432 4v5d6 ~a\n" (result->string (resolve-all d b) #:percent? #t))
;;   (format #t "1g+2432 4v5d6 ~a\n" (result->string (resolve-all e b) #:percent? #t)))

;; test fibers

(run-fibers
 (λ () 
   (let* ((a (gen-actions 3 6))
          (b (gen-actions 4 6))
          (r (result->string (resolve-all a b)))
          (s (result->string (resolve-all-fan-out a b))))
     (test-assert (equal? r s))))
 #:drain? #t)

(test-end "tests")

(define-syntax-rule (benchmark e ...)
  (let ((now (tms:clock (times))))
    (begin e ...)
    (format #t "~,2fs ~a\n" (/ (- (tms:clock (times)) now) 1e9) (quote e ...))))

(define (-bench)
  ;; 0.05s
  (benchmark (gen 6 6))
  ;; 0.18s
  (benchmark (gen-actions 6 6))
  ;;
  (benchmark (gen-actions 6 6 mod-2432! mod-1g!))
  ;; 0.28s
  (benchmark (resolve-all (gen-actions 4 6) (gen-actions 4 6)))

  ;; 2.01s
  (benchmark (resolve-all (gen-actions 4 6) (gen-actions 5 6)))
  ;; 0.65s
  (benchmark (resolve-all-fan-out (gen-actions 4 6) (gen-actions 5 6)))

  ;; 16.67s
  ;; (benchmark (resolve-all (gen-actions 5 6) (gen-actions 5 6)))
  ;; 5.18s
  ;; (benchmark (resolve-all-fan-out (gen-actions 5 6) (gen-actions 5 6)))

  ;; 42.56s
  ;; (benchmark (resolve-all-fan-out (gen-actions 5 6) (gen-actions 6 6)))

  ;; 335.60s
  ;; (benchmark (resolve-all-fan-out (gen-actions 6 6) (gen-actions 6 6)))

  )

(define (bench args)
  (format #t "%%%% Starting benchmarks ~a\n" args)
  (run-fibers -bench #:drain? #t))

(define (plot-resolve title a b)
  ;; sorting by wins illustrates the small number of groups all rolls will fall into
  ;; TODO check if tie,loss,part are all equal within each group
  (let ((rs (list->vector (sort (resolve-each-fan-out a b) less-win))))
    (with-output-to-file (format #f "~a.dat" title)
      (λ ()
        (vector-for-each
         (λ (i r)
           (format #t "~f ~f\n" (/ (+ 1 i) (vector-length rs)) (r #:percent? #t)))
         rs)))))

(define (-plot)
  (define n3d6 (gen-actions 3 6))
  (define n4d6 (gen-actions 4 6))
  (define n4d6-mod2432 (gen-actions 4 6 mod-2432!))
  (define n4d6-mod1g (gen-actions 4 6 mod-1g!))
  (define n4d6-all (gen-actions 4 6 mod-2432! mod-1g!))
  (define n5d6 (gen-actions 5 6))
  (plot-resolve "n4v3d6" n4d6 n3d6)
  (plot-resolve "n4v4d6" n4d6 n4d6)
  (plot-resolve "n4v5d6" n4d6 n5d6)
  (plot-resolve "mod2432-n4v4d6" n4d6-mod2432 n4d6)
  (plot-resolve "mod2432-n4v5d6" n4d6-mod2432 n5d6)
  (plot-resolve "mod1g-n4v4d6" n4d6-mod1g n4d6)
  (plot-resolve "mod1g-n4v5d6" n4d6-mod1g n5d6)
  (plot-resolve "modall-n4v5d6" n4d6-all n5d6)
  )

(define (plot args)
  (format #t "%%%% Starting plots ~a\n" args)
  (run-fibers -plot #:drain? #t))
